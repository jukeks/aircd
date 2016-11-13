package main

import (
	"aircd/protocol"
	"fmt"
	"log"
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

func NewUser(server *Server, conn net.Conn) *User {
	u := new(User)
	u.server = server
	u.lastPong = time.Now()
	u.conn = NewIrcConnection(u, conn)

	return u
}

func (user *User) hostmask() string {
	return fmt.Sprintf("%s!%s@%s", user.nick, user.username, user.hostname)
}

func (user *User) Close() {
	user.server.remove_user(user)
	user.conn.Close()
}

func (user *User) send_message(message protocol.IrcMessage) {
	user.conn.Send(message.Serialize())
}

func (user *User) send_message_from(from string, message protocol.IrcMessage) {
	user.conn.Send(fmt.Sprintf(":%s %s", from, message.Serialize()))
}

func (user *User) send_motd() {
	user.conn.Send(fmt.Sprintf(":%s 375 %s :- %s Message of the day - ",
		user.server.id, user.nick, user.server.id))

	for _, line := range user.server.get_motd() {
		user.conn.Send(fmt.Sprintf(":%s 372 %s :- %s",
			user.server.id, user.nick, line))
	}

	user.conn.Send(fmt.Sprintf(":%s 376 %s :End of /MOTD command.",
		user.server.id, user.nick))
}

func (user *User) send_users(users []string, channel string) {
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

func (user *User) Handle_message(message protocol.IrcMessage) {
	switch message.GetType() {
	case protocol.PONG:
		user.lastPong = time.Now()
	case protocol.NICK:
		msg := message.(protocol.NickMessage)
		user.server.handle_nick_change(user, msg.Nick)
	case protocol.USER:
		msg := message.(protocol.UserMessage)
		user.realname = msg.Realname
		user.username = msg.Username
		log.Printf("%s is %s!%s@%s",
			user.realname, user.nick, user.username, user.hostname)
	case protocol.JOIN:
		msg := message.(protocol.JoinMessage)
		user.server.handle_join(user, msg)
	case protocol.PART:
		msg := message.(protocol.PartMessage)
		user.server.handle_part(user, msg)
	case protocol.PRIVATE:
		msg := message.(protocol.PrivateMessage)
		user.server.handle_private_message(user, msg)
	case protocol.QUIT:
		user.Close()
		log.Printf("%s has quit.", user.nick)
	default:
		log.Printf("%s sent unknown message: %s", user.nick, message.Serialize())
	}
}
