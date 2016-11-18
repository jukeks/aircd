package main

import (
	"channeld/protocol"
	"fmt"
	"net"
	"time"
)

type User struct {
	nick     string
	username string
	realname string
	hostname string

	lastPong   time.Time
	registered bool

	server *Server
	conn   *IrcConnection
}

func NewUser(server *Server, conn net.Conn, incoming chan ClientAction) *User {
	u := new(User)
	u.server = server
	u.lastPong = time.Now()
	u.conn = NewIrcConnection(u, conn, incoming)

	return u
}

func (user *User) hostmask() string {
	return fmt.Sprintf("%s!%s@%s", user.nick, user.username, user.hostname)
}

func (user *User) Close() {
	user.conn.Close()
}

func (user *User) sendMessage(message protocol.IrcMessage) {
	user.conn.Send(message.Serialize())
}

func (user *User) sendSerializedMessage(message string) {
	user.conn.Send(message)
}

func getSerializedMessageFrom(from string,
	message protocol.IrcMessage) string {
	return fmt.Sprintf(":%s %s", from, message.Serialize())
}

func (user *User) sendMessageFrom(from string, message protocol.IrcMessage) {
	user.conn.Send(getSerializedMessageFrom(from, message))
}

func (user *User) sendMotd() {
	user.conn.Send(fmt.Sprintf(":%s 375 %s :- %s Message of the day - ",
		user.server.id, user.nick, user.server.id))

	for _, line := range user.server.get_motd() {
		user.conn.Send(fmt.Sprintf(":%s 372 %s :- %s",
			user.server.id, user.nick, line))
	}

	user.conn.Send(fmt.Sprintf(":%s 376 %s :End of /MOTD command.",
		user.server.id, user.nick))
}

func (user *User) sendUsers(users []string, channel string) {
	template := fmt.Sprintf(":%s 353 %s @ %s :",
		user.server.id, user.nick, channel)

	buff := ""
	for _, u := range users {
		if len(template)+len(buff)+len(u)+1 > 510 {
			user.conn.Send(fmt.Sprintf("%s%s", template, buff))
			buff = ""
		}

		buff = fmt.Sprintf("%s %s", u, buff)
	}

	user.conn.Send(fmt.Sprintf("%s%s", template, buff))

	user.conn.Send(fmt.Sprintf(":%s 366 %s :End of /NAMES list",
		user.server.id, user.nick))
}
