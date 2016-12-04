package server

import (
	"channeld/protocol"
	"fmt"
)

type User struct {
	nick     string
	username string
	realname string
	hostname string

	conn     *protocol.IrcConnection
}

func NewUser(nick, username, realname, hostname string,
	conn *protocol.IrcConnection) *User {
	u := new(User)
	u.nick = nick
	u.username = username
	u.realname = realname
	u.hostname = hostname
	u.conn = conn

	return u
}

func (user *User) hostmask() string {
	return fmt.Sprintf("%s!%s@%s", user.nick, user.username, user.hostname)
}

func (user *User) close() {
	user.conn.Close()
}
