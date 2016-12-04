package server

import (
	"channeld/protocol"
	"channeld/config"
	"channeld/channel"
	"log"
	"fmt"
)

func (server *Server) nickAvailable(nick string) bool {
	for _, user := range server.users {
		if user.nick == nick {
			return false
		}
	}

	return true
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

func (server *Server) handleNewUser(
	action protocol.ConnectionInitiationAction) {
	nickMsg := action.NickMessage
	userMsg := action.UserMessage
	if server.nickAvailable(nickMsg.Nick) {
		user := NewUser(nickMsg.Nick, userMsg.Username, userMsg.Realname,
						action.Hostname, action.Conn)
		server.addUser(action.Conn, user)
		server.sendMotd(action.Conn, user.nick)
		action.Conn.SendMessage(protocol.PingMessage{"12345"})

		action.ResponseChan <- protocol.ConnectionInitiationActionResponse{true,
			protocol.NO_ERROR}
	} else {
		action.ResponseChan <- protocol.ConnectionInitiationActionResponse{false,
			protocol.NICK_IN_USE}
	}
}

func (server *Server) handleMessage(action protocol.ClientAction) {
	message := action.Message
	conn := action.Connection
	user := server.getUserByConn(conn)

	if message == nil {
		server.removeUser(conn, user)
		log.Printf("%s has quit.", user.nick)
		return
	}

	if !isPrivateMessage(message) && isChannelMessage(message) {
		server.handleChannelMessage(user, action)
		return
	}

	switch message.GetType() {
	case protocol.PRIVATE:
		msg := message.(protocol.PrivateMessage)
		targetUser := server.getUserByName(msg.Target)
		if targetUser == nil {
			return
		}

		targetUser.conn.SendMessageFrom(user.hostmask(), action.Message)
	case protocol.PONG:
		//user.lastPong = time.Now()
	case protocol.NICK:
		msg := message.(protocol.NickMessage)
		server.handleNickChange(user, msg.Nick)
	case protocol.USER:
		log.Printf("")
	case protocol.QUIT:
		server.removeUser(conn, user)
		log.Printf("%s has quit.", user.nick)
	default:
		log.Printf("%s sent unknown message: %s", user.nick,
			message.Serialize())
	}
}

func (server *Server) handleChannelMessage(user *User,
	action protocol.ClientAction) {
	msg := action.Message.(protocol.ChannelMessage)

	c := server.getChannel(msg.GetTarget())
	if c == nil && msg.GetType() == protocol.JOIN {
		c = server.addChannel(msg.GetTarget())
	}

	if c == nil {
		return
	}

	c.Incoming <- channel.ChannelAction{user.hostmask(), user.nick,
		user.conn, msg}
}

func (server *Server) handleNickChange(user *User, nick string) {
	if !server.nickAvailable(nick) {
		log.Printf("Nick %s already in use", nick)
		id := config.Config.ServerID
		msg := protocol.NumericMessage{id, 433, nick,
			"Nick name is already in use."}
		user.conn.SendMessage(msg)
		return
	}

	log.Printf("%s changed nick to %s", user.nick, nick)
	user.nick = nick
	// TODO HANDLE CHANNELS
}

func (server *Server) addUser(conn *protocol.IrcConnection, user *User) {
	server.users[conn] = user

	log.Printf("Server has %d users", len(server.users))
}

func (server *Server) removeUser(conn *protocol.IrcConnection, user *User) {
	user.close()

	/* TODO FIX
	for _, c := range server.channels {
		//c.removeUser(user)
	}*/

	delete(server.users, conn)
}

func (server *Server) getUserByConn(conn *protocol.IrcConnection) *User {
	return server.users[conn]
}

func (server *Server) getUserByName(name string) *User {
	for _, user := range server.users {
		if user.nick == name {
			return user
		}
	}

	return nil
}

func (server *Server) getChannel(name string) *channel.Channel {
	return server.channels[name]
}

func (server *Server) addChannel(name string) *channel.Channel {
	c := channel.NewChannel(name)
	go c.Serve()

	server.channels[name] = c

	log.Printf("Added new channel: %s", name)

	return c
}

func (server *Server) getMotd() []string {
	return []string{
		fmt.Sprintf("Welcome to %s running", config.Config.ServerID),
		"     _                   _   _ ",
		" ___| |_ ___ ___ ___ ___| |_| |",
		"|  _|   | .'|   |   | -_| | . |",
		"|___|_|_|__,|_|_|_|_|___|_|___|",
		"                               ",
		"version 0.1.0.",
	}
}

func (server *Server) sendMotd(conn *protocol.IrcConnection, target string) {
	id := config.Config.ServerID
	conn.Send(fmt.Sprintf(":%s 375 %s :- %s Message of the day - ",
		id, target, id))

	for _, line := range server.getMotd() {
		conn.Send(fmt.Sprintf(":%s 372 %s :- %s", id, target, line))
	}

	conn.Send(fmt.Sprintf(":%s 376 %s :End of /MOTD command.", id, target))
}
