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
        user.send_motd()
        user.send_message(protocol.PingMessage{"12345"})
    } else {
        log.Printf("%s changed nick to %s", user.nick, nick)
        user.nick = nick
    }
}

func (server *Server) handle_join(joinedUser *User,
                                  message protocol.JoinMessage) {
    server.mutex.Lock()
    channel := server.get_channel(message.Target)
    if channel == nil {
        channel = server.add_channel(message.Target)
    }

    channel.add_user(joinedUser)
    users := channel.get_users()
    user_names := channel.get_user_names()

    server.mutex.Unlock()

    for _, channel_user := range users {
        channel_user.send_message_from(joinedUser.hostmask(), message)
    }

    joinedUser.send_users(user_names, message.Target)
}

func (server *Server) handle_part(partedUser *User,
                                  message protocol.PartMessage) {
    server.mutex.Lock()
    channel := server.get_channel(message.Target)
    if channel == nil {
        server.mutex.Unlock()
        return
    }

    users := channel.get_users()
    channel.remove_user(partedUser)
    server.mutex.Unlock()

    for _, channel_user := range users {
        channel_user.send_message_from(partedUser.hostmask(), message)
    }
}

func (server *Server) handle_private_message(sendingUser *User,
                                     message protocol.PrivateMessage) {
    server.mutex.Lock()
    channel := server.get_channel(message.Target)
    if channel == nil {
        server.mutex.Unlock()
        return
    }

    users := channel.get_users()
    server.mutex.Unlock()

    for _, channel_user := range users {
        if channel_user == sendingUser {
            continue
        }

        channel_user.send_message_from(sendingUser.hostmask(), message)
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

    for _, c := range server.channels {
        c.remove_user(user)
    }
}

func (server *Server) get_channel(name string) *Channel {
    for _, c := range server.channels {
        if c.name == name {
            return c
        }
    }

    return nil
}

func (server *Server) add_channel(name string,) *Channel {
    c := NewChannel(name)
    server.channels = append(server.channels, c)

    return c
}

func (server *Server) get_motd() []string {
    return []string{
        "moi moi",
        "terve terve",
    }
}



