package main

import (
    "net"
    "strings"
    "time"
)

type User struct {
    server *Server
    conn net.Conn

    nick string
    username string
    realname string
    hostname string

    lastPong time.Time
    registered bool
}

func NewUser(conn net.Conn) *User {
    u := new(User)
    u.conn = conn
    u.lastPong = time.Now()

    return u
}

func (user *User) handle_hostname() {
    split := strings.Split(user.conn.RemoteAddr().String(), ":")
    remote := split[0]

    names, _ := net.LookupAddr(remote)
    if len(names) > 0 {
        remote = names[0]
    }

    user.hostname = remote
}
