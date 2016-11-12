package main

import (
    "net"
    "bufio"
    "strings"
    "time"
    "log"
    "errors"
    "fmt"
    "aircd/protocol"
)

type User struct {
    server *Server
    conn net.Conn
    reader *bufio.Reader

    nick string
    username string
    realname string
    hostname string

    lastPong time.Time
    registered bool
}

func NewUser(server *Server, conn net.Conn) *User {
    u := new(User)
    u.server = server
    u.conn = conn
    u.reader = bufio.NewReader(conn)
    u.lastPong = time.Now()

    return u
}

func (user *User) serve() {
    defer user.conn.Close()

    user.handle_hostname()

    for {
        message, err := user.read_message()
        if err != nil {
            log.Printf("%s read failed: %v", user.nick, err)
            user.server.remove_user(user)
            return
        }

        user.server.handle_message(user, message)
    }
}

func (user *User) read_message() (protocol.IrcMessage, error) {
    line, err := user.reader.ReadString('\n')

    if err != nil || len(line) == 0 {
        if len(line) == 0 {
            return nil, errors.New("Empty line")
        }

        return nil, err
    }

    return protocol.ParseMessage(line[:len(line)-2]), nil
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
