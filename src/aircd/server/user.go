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

        user.handle_message(message)
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

func (user *User) send(message string) {
    buff := fmt.Sprintf("%s\r\n", message)
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

    log.Printf("Sent to %s: %s", user.nick, message)
}

func (user *User) send_message(message protocol.IrcMessage) {
    user.send(message.Serialize())
}

func (user *User) send_motd() {
    user.send(fmt.Sprintf(":%s 375 %s :- %s Message of the day - ",
                          user.server.id, user.nick, user.server.id))

    for _, line := range user.server.get_motd() {
        user.send(fmt.Sprintf(":%s 372 %s :- %s", user.server.id, user.nick, line))
    }

    user.send(fmt.Sprintf(":%s 376 %s :End of /MOTD command.", user.server.id, user.nick))
}

func (user *User) handle_message(message protocol.IrcMessage) {
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
        log.Printf("%s is %s!%s@%s", user.realname, user.nick, user.username,
                   user.hostname)

        user.send_motd()
        user.send_message(protocol.PingMessage{"12345"})

    case protocol.QUIT:
        user.conn.Close()
        user.server.remove_user(user)
        log.Printf("%s has quit.", user.nick)
    default:
    }
}
