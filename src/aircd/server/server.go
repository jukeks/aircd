package main

import (
	"aircd/protocol"
	"log"
	"net"
	"time"
)

type Server struct {
	id       string
	channels []*Channel
	users    []*User
	incoming chan ServerMessage
}

func NewServer(id string) *Server {
	s := new(Server)
	s.id = id
	s.channels = []*Channel{}
	s.users = []*User{}
	s.incoming = make(chan ServerMessage, 1000)

	return s
}

type ServerMessage struct {
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

func (server *Server) handle_message(action ServerMessage) {
	message := action.message
	user := action.user

	if message == nil {
		server.remove_user(user)
		log.Printf("%s has quit.", user.nick)
		return
	}

	switch message.GetType() {
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
	case protocol.JOIN:
		msg := message.(protocol.JoinMessage)
		server.handle_join(user, msg)
	case protocol.PART:
		msg := message.(protocol.PartMessage)
		server.handle_part(user, msg)
	case protocol.PRIVATE:
		msg := message.(protocol.PrivateMessage)
		server.handle_private_message(user, msg)
	case protocol.QUIT:
		server.remove_user(user)
		log.Printf("%s has quit.", user.nick)
	default:
		log.Printf("%s sent unknown message: %s", user.nick, message.Serialize())
	}
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

func (server *Server) handle_join(joinedUser *User,
	message protocol.JoinMessage) {
	channel := server.get_channel(message.Target)
	if channel == nil {
		channel = server.add_channel(message.Target)
	}

	channel.add_user(joinedUser)
	users := channel.get_users()
	user_names := channel.get_user_names()

	for _, channel_user := range users {
		channel_user.send_message_from(joinedUser.hostmask(), message)
	}

	joinedUser.send_users(user_names, message.Target)
}

func (server *Server) handle_part(partedUser *User,
	message protocol.PartMessage) {
	channel := server.get_channel(message.Target)
	if channel == nil {
		return
	}

	users := channel.get_users()
	channel.remove_user(partedUser)

	for _, channel_user := range users {
		channel_user.send_message_from(partedUser.hostmask(), message)
	}
}

func (server *Server) handle_private_message(sendingUser *User,
	message protocol.PrivateMessage) {
	channel := server.get_channel(message.Target)
	if channel == nil {
		return
	}

	for _, channel_user := range channel.get_users() {
		if channel_user == sendingUser {
			continue
		}

		channel_user.send_message_from(sendingUser.hostmask(), message)
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

func (server *Server) get_channel(name string) *Channel {
	for _, c := range server.channels {
		if c.name == name {
			return c
		}
	}

	return nil
}

func (server *Server) add_channel(name string) *Channel {
	c := NewChannel(name)
	server.channels = append(server.channels, c)

	return c
}

func (server *Server) get_motd() []string {
	return []string{
		"moi moi",
		"terve terve",
	}
}
