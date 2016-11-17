package protocol

import (
	"bufio"
	"fmt"
	"strings"
	"net"
	"time"
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
	NUMERIC

	UNKNOWN
)

type IrcMessage interface {
	GetType() MessageType
	Serialize() string
}

type ChannelMessage interface {
	IrcMessage
	GetTarget() string
}

/* -------------------------------------------------------------------------- */
type PingMessage struct {
	Token string
}

func (m PingMessage) GetType() MessageType {
	return PING
}

func (m PingMessage) Serialize() string {
	return fmt.Sprintf("PING :%s", m.Token)
}

/* -------------------------------------------------------------------------- */
type PongMessage struct {
	Token string
}

func (m PongMessage) GetType() MessageType {
	return PONG
}

func (m PongMessage) Serialize() string {
	return fmt.Sprintf("PONG :%s", m.Token)
}

/* -------------------------------------------------------------------------- */
type UnknownMessage struct {
	Message string
}

func (m UnknownMessage) GetType() MessageType {
	return UNKNOWN
}

func (m UnknownMessage) Serialize() string {
	return m.Message
}

/* -------------------------------------------------------------------------- */
type NickMessage struct {
	Nick string
}

func (m NickMessage) GetType() MessageType {
	return NICK
}

func (m NickMessage) Serialize() string {
	return fmt.Sprintf("NICK %s", m.Nick)
}

/* -------------------------------------------------------------------------- */
type UserMessage struct {
	Username string
	Realname string
	Hostname string
}

func (m UserMessage) GetType() MessageType {
	return USER
}

func (m UserMessage) Serialize() string {
	return ""
}

/* -------------------------------------------------------------------------- */
type PrivateMessage struct {
	Target  string
	Message string
}

func (m PrivateMessage) GetType() MessageType {
	return PRIVATE
}

func (m PrivateMessage) Serialize() string {
	return fmt.Sprintf("PRIVMSG %s :%s", m.Target, m.Message)
}

func (m PrivateMessage) GetTarget() string {
	return m.Target
}

/* -------------------------------------------------------------------------- */
type JoinMessage struct {
	Target string
}

func (m JoinMessage) GetType() MessageType {
	return JOIN
}

func (m JoinMessage) Serialize() string {
	return fmt.Sprintf("JOIN :%s", m.Target)
}

func (m JoinMessage) GetTarget() string {
	return m.Target
}

/* -------------------------------------------------------------------------- */
type PartMessage struct {
	Target string
}

func (m PartMessage) GetType() MessageType {
	return PART
}

func (m PartMessage) Serialize() string {
	return fmt.Sprintf("PART :%s", m.Target)
}

func (m PartMessage) GetTarget() string {
	return m.Target
}

/* -------------------------------------------------------------------------- */
type QuitMessage struct {
	Message string
}

func (m QuitMessage) GetType() MessageType {
	return QUIT
}

func (m QuitMessage) Serialize() string {
	return fmt.Sprintf("QUIT :%s", m.Message)
}

/* -------------------------------------------------------------------------- */
type NumericMessage struct {
	Source  string
	Code    int
	Target  string
	Message string
}

func (m NumericMessage) GetType() MessageType {
	return PRIVATE
}

func (m NumericMessage) Serialize() string {
	return fmt.Sprintf(":%s %d * %s :%s", m.Source, m.Code, m.Target,
		m.Message)
}

/* -------------------------------------------------------------------------- */
func ParseMessage(message string) IrcMessage {
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
		return UserMessage{username, realname, hostname}
	case "PRIVMSG":
		split = strings.SplitN(message, " ", 3)
		return PrivateMessage{split[1], split[2][1:]}
	case "JOIN":
		return JoinMessage{split[1]}
	case "PART":
		return PartMessage{split[1]}
	case "QUIT":
		return QuitMessage{split[1][1:]}
	default:
		return UnknownMessage{message}
	}
}

func WriteLine(conn net.Conn, message string) error {
	buff := fmt.Sprintf("%s\r\n", message)
	sent := 0

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	for sent < len(buff) {
		wrote, err := fmt.Fprintf(conn, buff[sent:])
		if err != nil || wrote == 0 {
			return err
		}

		sent += wrote
	}

	return nil
}

func ReadLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return line, err
	}

	line = line[:len(line)-2]
	return line, nil
}


