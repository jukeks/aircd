package channel

import (
	"channeld/config"
	"channeld/protocol"

	"log"
	"fmt"
)

type Channel struct {
	Name     string
	Incoming chan ChannelAction

	mode  string
	users []*ChannelUser
}

type ChannelAction struct {
	OriginHostMask string
	OriginNick     string
	OriginConn     *protocol.IrcConnection
	Message        protocol.IrcMessage
}

type ChannelUser struct {
	nick     string
	hostmask string
	conn     *protocol.IrcConnection
}

func NewChannel(name string) *Channel {
	c := new(Channel)
	c.Name = name
	c.users = []*ChannelUser{}

	c.Incoming = make(chan ChannelAction, 1000)

	return c
}

func (channel *Channel) Serve() {
	for {
		select {
		case action := <-channel.Incoming:
			channel.handleMessage(action)
		}

		if len(channel.users) == 0 {
			return
		}
	}
}

func (channel *Channel) handleMessage(action ChannelAction) {
	switch action.Message.GetType() {
	case protocol.PRIVATE:
		msg := action.Message.(protocol.PrivateMessage)
		channel.handlePrivateMessage(action, msg)
	case protocol.JOIN:
		msg := action.Message.(protocol.JoinMessage)
		channel.handleJoin(action, msg)
	case protocol.PART:
		msg := action.Message.(protocol.PartMessage)
		channel.handlePart(action, msg)
	case protocol.QUIT:
		msg := action.Message.(protocol.QuitMessage)
		channel.handleQuit(action, msg)
	default:
		log.Printf("Channel message not implemented: %v", action)
	}
}

func (channel *Channel) addUser(user *ChannelUser) {
	log.Printf("User %s joined channel %s", user.nick, channel.Name)
	channel.users = append(channel.users, user)
}

func (channel *Channel) removeUser(leavingUser *ChannelUser) {
	log.Printf("User %s left channel %s", leavingUser.nick, channel.Name)
	for i, user := range channel.users {
		if user.nick == leavingUser.nick {
			a := channel.users
			a[i] = a[len(a)-1]
			channel.users = a[:len(a)-1]

			log.Printf("Channel has %d users", len(channel.users))
			return
		}
	}
}

func (channel *Channel) getUserNames() []string {
	users := []string{}
	for _, u := range channel.users {
		users = append(users, u.nick)
	}

	return users
}

func (channel *Channel) getUserByNick(nick string) *ChannelUser {
	for _, u := range channel.users {
		if u.nick == nick {
			return u
		}
	}

	return nil
}

func (channel *Channel) handleJoin(action ChannelAction,
	message protocol.JoinMessage) {
	newUser := ChannelUser{action.OriginNick, action.OriginHostMask,
		action.OriginConn}
	channel.addUser(&newUser)

	serialized := protocol.GetSerializedMessageFrom(action.OriginHostMask,
		message)

	for _, user := range channel.users {
		user.conn.Send(serialized)
	}

	channel.sendUsers(newUser.hostmask, newUser.conn)
}

func (channel *Channel) handlePart(action ChannelAction,
	message protocol.PartMessage) {
	leavingUser := channel.getUserByNick(action.OriginNick)
	channel.removeUser(leavingUser)

	serialized := protocol.GetSerializedMessageFrom(action.OriginHostMask,
		message)

	for _, user := range channel.users {
		user.conn.Send(serialized)
	}

	leavingUser.conn.Send(serialized)
}

func (channel *Channel) handlePrivateMessage(action ChannelAction,
	message protocol.PrivateMessage) {
	serialized := protocol.GetSerializedMessageFrom(action.OriginHostMask,
		message)

	for _, user := range channel.users {
		if user.nick == action.OriginNick {
			continue
		}

		user.conn.Send(serialized)
	}
}

func (channel *Channel) handleQuit(action ChannelAction,
	message protocol.QuitMessage) {
	quitingUser := channel.getUserByNick(action.OriginNick)
	if quitingUser == nil {
		return
	}

	channel.removeUser(quitingUser)

	serialized := protocol.GetSerializedMessageFrom(action.OriginHostMask,
		message)

	for _, user := range channel.users {
		user.conn.Send(serialized)
	}
}

func (channel *Channel) sendUsers(target string, conn *protocol.IrcConnection) {
	serverId := config.Config.ServerID
	template := fmt.Sprintf(":%s 353 %s @ %s :",
		serverId, target, channel.Name)

	buff := ""
	for _, u := range channel.getUserNames() {
		if len(template)+len(buff)+len(u)+1 > 510 {
			conn.Send(fmt.Sprintf("%s%s", template, buff))
			buff = ""
		}

		buff = fmt.Sprintf("%s %s", u, buff)
	}

	conn.Send(fmt.Sprintf("%s%s", template, buff))

	conn.Send(fmt.Sprintf(":%s 366 %s :End of /NAMES list",
		serverId, target))
}
