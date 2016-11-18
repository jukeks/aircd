package main

import (
	"channeld/protocol"
	"log"
	"net"
	"time"
)

type Server struct {
	id       string
	channels map[string]*Channel
	users    []*User
	incoming chan ClientAction
}

func NewServer(id string) *Server {
	s := new(Server)
	s.id = id
	s.channels = make(map[string]*Channel)
	s.users = []*User{}
	s.incoming = make(chan ClientAction, 1000)

	return s
}

type ClientAction struct {
	user    *User
	message protocol.IrcMessage
}

func (server *Server) Serve() {
	listener, _ := net.Listen("tcp", ":6667")

	quit := make(chan bool)

	go server.serveUsers(quit)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		user := NewUser(server.id, conn, server.incoming)
		go user.conn.Serve()
	}

	quit <- true
}

func (server *Server) nickAvailable(nick string) bool {
	for _, user := range server.users {
		if user.nick == nick {
			return false
		}
	}

	return true
}

func (server *Server) serveUsers(quit chan bool) {
	for {
		select {
		case <-quit:
			return
		case action := <-server.incoming:
			server.handleMessage(action)
		}
	}
}

func isChannelMessage(message protocol.IrcMessage) bool {
	switch message.(type) {
	case protocol.ChannelMessage:
		return true
	default:
		return false
	}
}

func isPrivateMessage(message protocol.IrcMessage) bool {
	if message.GetType() != protocol.PRIVATE {
		return false
	}

	msg := message.(protocol.PrivateMessage)
	chan_type := msg.Target[0]
	if chan_type == '#' || chan_type == '!' {
		return false
	}

	return true
}

func (server *Server) handleMessage(action ClientAction) {
	message := action.message
	user := action.user

	if message == nil {
		server.removeUser(user)
		log.Printf("%s has quit.", user.nick)
		return
	}

	if !isPrivateMessage(message) && isChannelMessage(message) {
		server.handleChannelMessage(action)
		return
	}

	switch message.GetType() {
	case protocol.PRIVATE:
		msg := message.(protocol.PrivateMessage)
		target_user := server.getUser(msg.Target)
		if target_user == nil {
			return
		}

		target_user.sendMessageFrom(user.hostmask(), action.message)
	case protocol.PONG:
		user.lastPong = time.Now()
	case protocol.NICK:
		msg := message.(protocol.NickMessage)
		server.handleNickChange(user, msg.Nick)
	case protocol.USER:
		msg := message.(protocol.UserMessage)
		user.realname = msg.Realname
		user.username = msg.Username
		log.Printf("%s is %s!%s@%s",
			user.realname, user.nick, user.username, user.hostname)
	case protocol.QUIT:
		server.removeUser(user)
		log.Printf("%s has quit.", user.nick)
	default:
		log.Printf("%s sent unknown message: %s", user.nick, message.Serialize())
	}
}

func (server *Server) handleChannelMessage(action ClientAction) {
	msg := action.message.(protocol.ChannelMessage)

	channel := server.getChannel(msg.GetTarget())
	if channel == nil && msg.GetType() == protocol.JOIN {
		channel = server.addChannel(msg.GetTarget())
	}

	if channel == nil {
		return
	}

	channel.incoming <- action
}

func (server *Server) handleNickChange(user *User, nick string) {
	if !server.nickAvailable(nick) {
		log.Printf("Nick %s already in use", nick)
		msg := protocol.NumericMessage{server.id, 433, nick,
			"Nick name is already in use."}
		user.sendMessage(msg)
		return
	}

	if !user.registered {
		user.nick = nick
		log.Printf("New user: %s", nick)
		server.addUser(user)
		user.registered = true
		user.sendMotd(server.getMotd())
		user.sendMessage(protocol.PingMessage{"12345"})
	} else {
		log.Printf("%s changed nick to %s", user.nick, nick)
		user.nick = nick
	}
}

func (server *Server) addUser(user *User) {
	server.users = append(server.users, user)

	log.Printf("Server has %d users", len(server.users))
}

func (server *Server) removeUser(user *User) {
	user.Close()

	for _, c := range server.channels {
		c.removeUser(user)
	}

	for i, i_u := range server.users {
		if i_u.nick == user.nick {
			a := server.users
			a[i] = a[len(a)-1]
			server.users = a[:len(a)-1]
			log.Printf("Server has %d users", len(server.users))
			break
		}
	}
}

func (server *Server) getUser(name string) *User {
	for _, user := range server.users {
		if user.nick == name {
			return user
		}
	}

	return nil
}

func (server *Server) getChannel(name string) *Channel {
	return server.channels[name]
}

func (server *Server) addChannel(name string) *Channel {
	c := NewChannel(name)
	go c.Serve()

	server.channels[name] = c

	log.Printf("Added new channel: %s", name)

	return c
}

func (server *Server) getMotd() []string {
	return []string{
		"moi moi",
		"terve terve",
	}
}
