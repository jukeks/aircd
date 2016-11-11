package protocol

import (
    "strings"
    "strconv"
    //"fmt"
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
    getType() MessageType
}


type PingMessage struct {
    Token string
}

func (m PingMessage) getType() MessageType {
    return PING
}

type UnknownMessage struct {
    Message string
}

func (m UnknownMessage) getType() MessageType {
    return UNKNOWN
}

type NickMessage struct {
    Nick string
}

func (m NickMessage) getType() MessageType {
    return NICK
}

type UserMessage struct {
    Username string
    Realname string
    Mode uint8
}

func (m UserMessage) getType() MessageType {
    return USER
}

type PrivateMessage struct {
    Target string
    Message string
}

func (m PrivateMessage) getType() MessageType {
    return PRIVATE
}


func ParseMessage(message string) (IrcMessage) {
    split := strings.SplitN(message, " ", 2)
    switch command := split[0]; command {
        case "PING":
            return PingMessage{split[1][1:]}
        case "NICK":
            return NickMessage{split[1]}
        case "USER":
            split = strings.SplitN(message, " ", 5)
            if len(split) != 5 {
                return UnknownMessage{message}
            }

            username := split[1]
            realname := split[4][1:]
            mode, err := strconv.ParseInt(split[2], 10, 8)
            if err != nil || mode < 0 {
                return UnknownMessage{message}
            }

            return UserMessage{username, realname, uint8(mode)}
        case "PRIVMSG":
            split = strings.SplitN(message, " ", 3)
            return PrivateMessage{split[1], split[2][1:]}
        default:
            return UnknownMessage{message}
    }
}
