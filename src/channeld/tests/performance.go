package main

import (
	"bufio"
	"channeld/protocol"
	"channeld/server"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

var logger *log.Logger

const NUM_CHANNELS = 50

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	nick   string
	id     int
}

func NewClient(id int, nick string) *Client {
	c := new(Client)
	c.id = id
	c.nick = nick

	return c
}

func connect() (net.Conn, error) {
	// kernel listen backlog is roughly 128, need to randomize the arrival times
	// a bit to get everyone in.

	for i := 0; i < 3; i++ {
		time.Sleep(time.Duration(rand.Int31n(1000)) * time.Millisecond)
		conn, err := net.Dial("tcp", "127.0.0.1:6667")
		if err != nil {
			log.Printf("Connection failed: %v. Retrying", err)
			time.Sleep(time.Duration((rand.Int31n(6)+1)*500) * time.Millisecond)
			continue
		}

		return conn, err
	}

	return nil, nil
}

func (c *Client) join(channel string) {
	protocol.WriteLine(c.conn, fmt.Sprintf("JOIN %s", channel))
	readUntil(c.reader, fmt.Sprintf("JOIN :%s", channel))
}

func (c *Client) handshake() {
	conn, err := connect()
	if err != nil {
		panic(err)
	}

	if conn == nil {
		panic("failed to connect")
	}

	reader := bufio.NewReader(conn)

	c.conn = conn
	c.reader = reader

	protocol.WriteLine(conn, fmt.Sprintf("NICK %s", c.nick))
	protocol.WriteLine(conn, fmt.Sprintf("USER %s localhost localhost :Teppo", c.nick))
	readUntil(reader, "PING")
}

func readUntil(reader *bufio.Reader, until string) {
	for {
		line, err := protocol.ReadLine(reader)
		if err != nil {
			log.Printf("GOT EMPTY LINE %v", err)
			return
		}

		if strings.Contains(line, until) {
			return
		}
	}
}

func readerRoutine(id, numWriters int, done chan int, joined chan bool) {
	client := NewClient(1, fmt.Sprintf("reader%d", id))
	client.handshake()
	client.join(fmt.Sprintf("#testers%d", id))
	joined <- true

	expectedParts := numWriters / NUM_CHANNELS

	messages := 0
	parts := 0

	for {
		client.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := protocol.ReadLine(client.reader)
		if err != nil {
			log.Printf("Bailing: %s", err)
			break
		}

		messages += 1

		if strings.Contains(line, "PART ") {
			parts += 1
			if parts%20 == 0 {
				log.Printf("PART #%d received", parts)
			}
		}

		if parts == expectedParts {
			log.Printf("READ %d PART messages", expectedParts)
			break
		}
	}

	client.conn.Close()
	log.Printf("Read %d messages", messages)
	done <- messages
}

func writerRoutine(done chan bool, id int, name string, n int, joined, start chan bool) {
	client := NewClient(id, name)
	client.handshake()
	defer client.conn.Close()

	channel := fmt.Sprintf("#testers%d", id%NUM_CHANNELS)
	client.join(channel)
	joined <- true
	<-start

	go func() {
		for i := 0; i < n; i++ {
			protocol.WriteLine(client.conn,
				fmt.Sprintf("PRIVMSG %s :%s", channel, strings.Repeat("A", 460)))
		}

		protocol.WriteLine(client.conn, fmt.Sprintf("PART %s", channel))
	}()

	for {
		client.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		line, err := protocol.ReadLine(client.reader)
		if err != nil {
			continue
		}

		if strings.Contains(line, fmt.Sprintf(":%s!", name)) && strings.Contains(line, "PART") {
			break
		}

	}

	done <- true
}

func main() {
	runtime.GOMAXPROCS(8)
	s := server.NewServer("test.server.example.org")
	go s.Serve()

	numWriters := 1500
	numMessages := 10

	writersDone := make(chan bool, numWriters)
	messagesRead := make(chan int, NUM_CHANNELS)
	readersJoined := make(chan bool, NUM_CHANNELS)
	writersJoined := make(chan bool, numWriters)
	writersStart := make(chan bool, numWriters)

	// joining readers
	for i := 0; i < NUM_CHANNELS; i++ {
		go readerRoutine(i, numWriters, messagesRead, readersJoined)
	}
	for i := 0; i < NUM_CHANNELS; i++ {
		<-readersJoined
	}

	// joinin gwriters
	for i := 0; i < numWriters; i++ {
		func() {
			go writerRoutine(writersDone, i, fmt.Sprintf("writer%d", i),
				numMessages, writersJoined, writersStart)
		}()
	}
	for i := 0; i < numWriters; i++ {
		<-writersJoined
	}

	log.SetOutput(ioutil.Discard)

	start := time.Now()
	for i := 0; i < numWriters; i++ {
		writersStart <- true
	}

	for i := 0; i < numWriters; i++ {
		<-writersDone
	}

	messages := 0
	for i := 0; i < NUM_CHANNELS; i++ {
		messages += <-messagesRead
	}

	writesDone := time.Since(start)

	log.SetOutput(os.Stdout)
	log.Printf("Read %d messages in %s", messages, writesDone)

	//s.Quit()
}
