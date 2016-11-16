package main

import (
	"bufio"
	"channeld/protocol"
	"fmt"
	"log"
	"net"
	"strings"
)

var logger *log.Logger

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func handshake(nick string) *Client {
	conn, err := net.Dial("tcp", "localhost:6667")
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(conn)

	client := new(Client)
	client.conn = conn
	client.reader = reader

	protocol.WriteLine(conn, fmt.Sprintf("NICK %s", nick))
	protocol.WriteLine(conn, fmt.Sprintf("USER tester localhost localhost :Teppo"))
	read_until(reader, "PING")
	protocol.WriteLine(conn, "JOIN #testers")
	read_until(reader, "JOIN :#testers")

	return client
}

func read_until(reader *bufio.Reader, until string) {
	for {
		line, err := protocol.ReadLine(reader)
		if err != nil {
			panic(err)
		}

		log.Printf("READ LINE: %s", line)

		if strings.Contains(line, until) {
			log.Printf("GOT %s", line)
			return
		}
	}
}

func read_until_part(started chan bool) {
	client := handshake("tester1")
	defer client.conn.Close()

	started <- true

	log.Printf("Wrote to started")
	messages := 0
	for {
		line, err := protocol.ReadLine(client.reader)
		if err != nil {
			panic(err)
		}

		if strings.Contains(line, "PART :#testers") {
			log.Printf("Read %d messages before quit.", messages)
			return
		}

		messages += 1
	}
}

func write_10000_lines(started, done chan bool, to string) {
	client := handshake("tester2")
	defer client.conn.Close()

	quit := make(chan bool)

	go func() {
		for {
			select {
			case <-quit:
				return
			default:
			}

			_, err := protocol.ReadLine(client.reader)
			if err != nil {
				<-quit
				return
			}
		}
	}()

	log.Printf("Waiting for started")
	<-started
	log.Printf("Got started")

	for i := 0; i < 1000; i++ {
		protocol.WriteLine(client.conn,
			fmt.Sprintf("PRIVMSG %s :%s", to, strings.Repeat("A", 480)))
	}

	protocol.WriteLine(client.conn, "PART #testers")

	log.Printf("Writer done")
	quit <- true
	done <- true
	log.Printf("Done")
}

func main() {
	started := make(chan bool)
	done := make(chan bool)

	go write_10000_lines(started, done, "tester1")
	read_until_part(started)
	<-done
}
