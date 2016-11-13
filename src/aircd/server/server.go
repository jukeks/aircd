package main

import (
	"aircd/protocol"
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

	go server.serve_users(quit)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		user := NewUser(server, conn, server.incoming)
		go user.conn.Serve()
	}

	quit <- true
}

func (server *Server) nick_available(nick string) bool {
	for _, user := range server.users {
		if user.nick == nick {
			return false
		}
	}

	return true
}

func (server *Server) serve_users(quit chan bool) {
	for {
		select {
		case <-quit:
			return
		case action := <-server.incoming:
			server.handle_message(action)
		}
	}
}

func is_channel_message(message protocol.IrcMessage) bool {
	switch message.(type) {
	case protocol.ChannelMessage:
		return true
	default:
		return false
	}
}

func is_private_message(message protocol.IrcMessage) bool {
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

func (server *Server) handle_message(action ClientAction) {
	message := action.message
	user := action.user

	if message == nil {
		server.remove_user(user)
		log.Printf("%s has quit.", user.nick)
		return
	}

	if !is_private_message(message) && is_channel_message(message) {
		server.handle_channel_message(action)
		return
	}

	switch message.GetType() {
	case protocol.PRIVATE:
		msg := message.(protocol.PrivateMessage)
		target_user := server.get_user(msg.Target)
		if user == nil {
			return
		}

		target_user.send_message_from(user.hostmask(), action.message)
	case protocol.PONG:
		user.lastPong = time.Now()
	case protocol.NICK:
		msg := message.(protocol.NickMessage)
		server.handle_nick_change(user, msg.Nick)
	case protocol.USER:
		msg := message.(protocol.UserMessage)
		user.realname = msg.Realname
		user.username = msg.Username
		log.Printf("%s is %s!%s@%s",
			user.realname, user.nick, user.username, user.hostname)
	case protocol.QUIT:
		server.remove_user(user)
		log.Printf("%s has quit.", user.nick)
	default:
		log.Printf("%s sent unknown message: %s", user.nick, message.Serialize())
	}
}

func (server *Server) handle_channel_message(action ClientAction) {
	msg := action.message.(protocol.ChannelMessage)

	channel := server.get_channel(msg.GetTarget())
	if channel == nil && msg.GetType() == protocol.JOIN {
		channel = server.add_channel(msg.GetTarget())
	}

	if channel == nil {
		return
	}

	channel.incoming <- action
}

func (server *Server) handle_nick_change(user *User, nick string) {
	if !server.nick_available(nick) {
		log.Printf("Nick %s already in use", nick)
		msg := protocol.NumericMessage{server.id, 433, nick,
			"Nick name is already in use."}
		user.send_message(msg)
		return
	}

	if !user.registered {
		user.nick = nick
		log.Printf("New user: %s", nick)
		server.add_user(user)
		user.registered = true
		user.send_motd()
		user.send_message(protocol.PingMessage{"12345"})
	} else {
		log.Printf("%s changed nick to %s", user.nick, nick)
		user.nick = nick
	}
}

func (server *Server) add_user(user *User) {
	server.users = append(server.users, user)

	log.Printf("Server has %d users", len(server.users))
}

func (server *Server) remove_user(user *User) {
	user.Close()

	for _, c := range server.channels {
		c.remove_user(user)
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

func (server *Server) get_user(name string) *User {
	for _, user := range server.users {
		if user.nick == name {
			return user
		}
	}

	return nil
}

func (server *Server) get_channel(name string) *Channel {
	return server.channels[name]
}

func (server *Server) add_channel(name string) *Channel {
	c := NewChannel(name)
	go c.Serve()

	server.channels[name] = c

	log.Printf("Added new channel: %s", name)

	return c
}

func (server *Server) get_motd() []string {
	return []string{
		"moi moi",
		"terve terve",
	}
}
