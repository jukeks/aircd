package main

import (
    "net"
    "strings"
    "time"
    "log"
    "fmt"
    "aircd/protocol"
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

func (user *User) send(message protocol.IrcMessage) {
    buff := fmt.Sprintf("%s\r\n", message.Serialize())

    sent := 0
    for sent < len(buff) {
        wrote, err :=  fmt.Fprintf(user.conn, buff[sent:])
        if err != nil || wrote == 0 {
            log.Printf("Error writing socket %v", err)
            user.conn.Close()
            return
        }

        sent += wrote
    }

    log.Printf("Sent to %s: %s", user.nick, message.Serialize())
}