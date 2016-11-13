package main

import (
	"aircd/protocol"
	"bufio"
	"errors"
	"fmt"
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

	outgoing chan string
	quit     chan bool
}

func NewIrcConnection(user *User, conn net.Conn) *IrcConnection {
	c := new(IrcConnection)

	c.user = user
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.mutex = sync.Mutex{}

	c.outgoing = make(chan string, 100)
	c.quit = make(chan bool)

	return c
}

func (conn *IrcConnection) Get_hostname() string {
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
	}
}

func (conn *IrcConnection) Serve() {
	defer conn.user.Close()

	go conn.writer_routine()

	conn.user.hostname = conn.Get_hostname()

	for {
		message, err := conn.read_message()
		if err != nil {
			log.Printf("%s read failed: %v", conn.user.nick, err)
			return
		}

		conn.user.Handle_message(message)
	}
}

func (conn *IrcConnection) Send(msg string) {
	conn.outgoing <- msg
}

func (conn *IrcConnection) writer_routine() {
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
	buff := fmt.Sprintf("%s\r\n", message)
	sent := 0

	conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	for sent < len(buff) {
		wrote, err := fmt.Fprintf(conn.conn, buff[sent:])
		if err != nil || wrote == 0 {
			log.Printf("Error writing socket %v", err)
			conn.user.Close()
			return
		}

		sent += wrote
	}

	log.Printf("Sent to %s: %s", conn.user.nick, message)
}

func (conn *IrcConnection) read_message() (protocol.IrcMessage, error) {
	line, err := conn.reader.ReadString('\n')

	if err != nil || len(line) == 0 {
		if len(line) == 0 {
			return nil, errors.New("Empty line")
		}

		return nil, err
	}

	line = line[:len(line)-2]

	log.Printf("User %s sent: %s", conn.user.nick, line)

	return protocol.ParseMessage(line), nil
}
