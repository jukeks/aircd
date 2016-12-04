package protocol

import (
	"fmt"
	"strings"
	"net"
	"bufio"
	"time"
)

func GetSerializedMessageFrom(from string,
	message IrcMessage) string {
	return fmt.Sprintf(":%s %s", from, message.Serialize())
}

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
