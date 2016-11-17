package main

import (
	"bufio"
	"channeld/protocol"
	"fmt"
	"log"
	"net"
	"time"
	"strings"
	"math/rand"
)

var logger *log.Logger

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func connect() (net.Conn, error) {
	// kernel listen backlog is roughly 128, need to randomize the arrival times
	// a bit to get everyone in.

	for i := 0; i < 3; i++ {
		time.Sleep(time.Duration(rand.Int31n(1000)) * time.Millisecond)
		conn, err := net.Dial("tcp", "127.0.0.1:6667")
		if err != nil {
			log.Printf("Connection failed: %v. Retrying", err)
			time.Sleep(time.Duration((rand.Int31n(6) +1) * 500) * time.Millisecond)
			continue
		}

		return conn, err
	}

	return nil, nil
}

func handshake(nick string) *Client {
	conn, err := connect()
	if err != nil {
		panic(err)
	}

	if conn == nil {
		panic("failed to connect")
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
			log.Printf("GOT EMPTY LINE %v", err)
			return
		}

		if strings.Contains(line, until) {
			return
		}
	}
}

func read_until_part(n int, done chan bool) {
	client := handshake("tester1")

	go func() {
		messages := 0
		parts := 0

		defer func() {
			client.conn.Close()
			log.Printf("Read %d messages", messages)
			done <- true
		}()

		for {
			client.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			line, err := protocol.ReadLine(client.reader)
			if err != nil {
				log.Printf("Bailing: %s", err)
				return
			}

			if strings.Contains(line, "PART ") {
				parts += 1
				if parts % 20 == 0 {
					log.Printf("PART #%d received", parts)
				}
			}

			if parts == n {
				log.Printf("READ %d PART messages", n)
				return
			}

			messages += 1
		}
	}()
}

func write_n_lines(checkpoint1, checkpoint2, done chan bool, name string, n int) {
	client := handshake(name)
	defer client.conn.Close()

	/*
	checkpoint1 <- true
	<-checkpoint2*/

	go func() {
		for i := 0; i < n; i++ {
			protocol.WriteLine(client.conn,
				fmt.Sprintf("PRIVMSG tester1 :%s", strings.Repeat("A", 460)))
		}

		protocol.WriteLine(client.conn, "PART #testers")
	}()

	for {
		client.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
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
	n := 500

	done := make(chan bool, n + 1)
	checkpoint1 := make(chan bool, n)
	checkpoint2 := make(chan bool, n)

	read_until_part(n, done)

	for i := 2; i < n + 2; i++ {
		func() {
			go write_n_lines(checkpoint1, checkpoint2, done, fmt.Sprintf("tester%d", i), 5)
		}()
	}

	/*
	for i := 0; i < n; i++ {
		<-checkpoint1
	}
	log.Printf("Everyone joined")
	for i := 0; i < n; i++ {
		checkpoint2 <- true
	}*/
	for i := 0; i < n; i++ {
		<-done
	}

	<-done
}
