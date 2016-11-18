package main

import (
	"bufio"
	"channeld/protocol"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type IrcConnection struct {
	user   *User
	conn   net.Conn
	reader *bufio.Reader

	mutex  sync.Mutex
	closed bool

	messageCounter uint32
	counterReseted time.Time

	incoming chan ClientAction
	outgoing chan string
	quit     chan bool
}

func NewIrcConnection(user *User, conn net.Conn,
	incoming chan ClientAction) *IrcConnection {
	c := new(IrcConnection)

	c.user = user
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.mutex = sync.Mutex{}

	c.counterReseted = time.Now()

	c.incoming = incoming
	c.outgoing = make(chan string, 1000)
	c.quit = make(chan bool, 2)

	return c
}

func (conn *IrcConnection) GetHostname() string {
	split := strings.Split(conn.conn.RemoteAddr().String(), ":")
	remote := split[0]

	names, _ := net.LookupAddr(remote)
	if len(names) > 0 {
		remote = names[0]
	}

	return remote
}

func (conn *IrcConnection) Close() {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if !conn.closed {
		conn.closed = true

		conn.conn.Close()
		conn.quit <- true
		conn.quit <- true
	}
}

func (conn *IrcConnection) checkCounter() bool {
	conn.messageCounter += 1

	if conn.messageCounter > 10 {
		return false
	}

	if time.Now().After(conn.counterReseted.Add(10 * time.Second)) {
		conn.messageCounter = 0
		conn.counterReseted = time.Now()
	}

	return true
}

func (conn *IrcConnection) Serve() {
	go conn.writerRoutine()

	conn.user.hostname = conn.GetHostname()

	for {
		select {
		case <-conn.quit:
			return
		default:
		}

		message, err := conn.readMessage()
		if err != nil {
			log.Printf("%s read failed: %v", conn.user.nick, err)
			conn.incoming <- ClientAction{conn.user, nil}
			return
		}

		if !conn.checkCounter() {
			log.Printf("Client flooding %d messages in %s",
					   conn.messageCounter,
					   time.Since(conn.counterReseted).String())
			conn.incoming <- ClientAction{conn.user, nil}
			return
		}

		conn.incoming <- ClientAction{conn.user, message}
	}
}

func (conn *IrcConnection) Send(msg string) {
	select {
	case conn.outgoing <- msg:
		return
	default:
		log.Printf("Client %s queue is full. Closing.", conn.user.nick)
		conn.user.Close()
	}
}

func (conn *IrcConnection) writerRoutine() {
	for {
		select {
		case msg := <-conn.outgoing:
			conn.write(msg)
		case <-conn.quit:
			return
		}
	}
}

func (conn *IrcConnection) write(message string) {
	err := protocol.WriteLine(conn.conn, message)
	if err != nil {
		log.Printf("Error writing socket %v", err)
		conn.incoming <- ClientAction{conn.user, nil}
		return
	}

	log.Printf("Sent to %s: %s", conn.user.nick, message)
}

func (conn *IrcConnection) readMessage() (protocol.IrcMessage, error) {
	line, err := protocol.ReadLine(conn.reader)
	if err != nil {
		return nil, err
	}

	log.Printf("User %s sent: %s", conn.user.nick, line)
	return protocol.ParseMessage(line), nil
}
