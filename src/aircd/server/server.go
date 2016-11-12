package main

import (
    "sync"
    "log"
    "aircd/protocol"
)

type Server struct {
    id string
    channels []*Channel
    users []*User
    mutex sync.Mutex
}

func (server *Server) nick_available(nick string) bool {
    for _, user := range server.users {
        if user.nick == nick {
            return false
        }
    }

    return true
}

func (server *Server) handle_nick_change(user *User, nick string) {
    server.mutex.Lock()
    defer server.mutex.Unlock()

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
    } else {
        log.Printf("%s changed nick to %s", user.nick, nick)
        user.nick = nick
    }
}

func (server *Server) add_user(user *User) {
    server.users = append(server.users, user)

    log.Printf("Server has %d users", len(server.users))
}

func (server *Server) remove_user(user *User) {
    server.mutex.Lock()
    defer server.mutex.Unlock()

    for i, i_u := range server.users {
        if i_u.nick == user.nick {
            a := server.users
            a[i] = a[len(a)-1]
            server.users = a[:len(a)-1]
            log.Printf("Server has %d users", len(server.users))
            return
        }
    }
}

func (server *Server) get_motd() []string {
    return []string{
        "moi moi",
        "terve terve",
    }
}



