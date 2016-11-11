package protocol

import (
    "strings"
)

type MessageType int

const (
    JOIN MessageType = iota
    PART
    QUIT
    PRIVATE
    TOPIC
    TOPIC_REPLY
    NICK
    USER
    PING
    PONG

    UNKNOWN
)

type IrcMessage interface {
    GetType() MessageType
}


type PingMessage struct {
    Token string
}

func (m PingMessage) GetType() MessageType {
    return PING
}

type PongMessage struct {
    Token string
}

func (m PongMessage) GetType() MessageType {
    return PONG
}

type UnknownMessage struct {
    Message string
}

func (m UnknownMessage) GetType() MessageType {
    return UNKNOWN
}

type NickMessage struct {
    Nick string
}

func (m NickMessage) GetType() MessageType {
    return NICK
}

type UserMessage struct {
    Username string
    Realname string
    Hostname string
    Mode uint8
}

func (m UserMessage) GetType() MessageType {
    return USER
}

type PrivateMessage struct {
    Target string
    Message string
}

func (m PrivateMessage) GetType() MessageType {
    return PRIVATE
}


func ParseMessage(message string) (IrcMessage) {
    split := strings.SplitN(message, " ", 2)
    switch command := split[0]; command {
        case "PONG":
            return PongMessage{split[1][1:]}
        case "NICK":
            return NickMessage{split[1]}
        case "USER":
            split = strings.SplitN(message, " ", 5)
            if len(split) != 5 {
                return UnknownMessage{message}
            }

            username := split[1]
            hostname := split[2]
            realname := split[4][1:]
            return UserMessage{username, realname, hostname, 0}
        case "PRIVMSG":
            split = strings.SplitN(message, " ", 3)
            return PrivateMessage{split[1], split[2][1:]}
        default:
            return UnknownMessage{message}
    }
}
