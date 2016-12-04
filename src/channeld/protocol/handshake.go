package protocol

import (
	"log"
)

type handshake struct {
	hostname string
	nickMessage NickMessage
	userMessage UserMessage
	nickReceived bool
	userReceived bool
	messagesRead int
	nickRetries int
	newClients chan ConnectionInitiationAction
	responseChan chan ConnectionInitiationActionResponse

	conn *IrcConnection
}

func newHandshake(conn *IrcConnection,
	newClients chan ConnectionInitiationAction, hostname string) *handshake {
	hs := new(handshake)
	hs.nickReceived = false
	hs.userReceived = false
	hs.messagesRead = 0
	hs.nickRetries = 0
	hs.responseChan = make(chan ConnectionInitiationActionResponse, 1)
	hs.hostname = hostname
	hs.newClients = newClients
	hs.conn = conn

	return hs
}

func (hs *handshake) readMessages() bool {
	for (!hs.nickReceived || !hs.userReceived) && hs.messagesRead < 4 {
		select {
		case <-hs.conn.quit:
			return false
		default:
		}

		message, err := hs.conn.readMessage()
		if err != nil {
			log.Printf("%v read failed: %v", hs.conn, err)
			return false
		}
		hs.messagesRead += 1

		if message.GetType() == USER {
			hs.userMessage = message.(UserMessage)
			hs.userReceived = true
		} else if message.GetType() == NICK {
			hs.nickMessage = message.(NickMessage)
			hs.nickReceived = true
		}
	}

	return hs.nickReceived && hs.userReceived
}

func (hs *handshake) register() bool {
	if hs.nickReceived && hs.userReceived {
		hs.newClients <- ConnectionInitiationAction{hs.userMessage,
			hs.nickMessage, hs.hostname, hs.conn, hs.responseChan}
		response := <-hs.responseChan

		if !response.Success {
			hs.conn.write(response.Reply.Serialize())
			hs.nickReceived = false
			hs.messagesRead = 0
			return false
		}

		return true
	}

	return false
}

func (conn *IrcConnection) handshake(
	newClients chan ConnectionInitiationAction) bool {
	hs := newHandshake(conn, newClients, conn.getHostname())

	for hs.nickRetries < 3 {
		ok := hs.readMessages()
		if !ok {
			return false
		}

		ok = hs.register()
		if !ok {
			hs.nickRetries += 1
			continue
		}

		return true
	}

	return false
}
