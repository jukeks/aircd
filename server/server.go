package server

import (
	"github.com/jukeks/channeld/channel"
	"github.com/jukeks/channeld/config"
	"github.com/jukeks/channeld/protocol"

	"log"
	"net"
	"time"
)

type Server struct {
	channels map[string]*channel.Channel
	users    map[*protocol.IrcConnection]*User
	incoming chan protocol.ClientAction
	newUsers chan protocol.ConnectionInitiationAction
	quit     chan bool
}

func NewServer(id string) *Server {
	s := new(Server)
	s.channels = make(map[string]*channel.Channel)
	s.users = make(map[*protocol.IrcConnection]*User)
	s.incoming = make(chan protocol.ClientAction, 1000)
	s.newUsers = make(chan protocol.ConnectionInitiationAction)
	s.quit = make(chan bool)

	config.Config.ServerID = id

	return s
}

func (server *Server) Quit() {
	server.quit <- true
	server.quit <- true
}

func (server *Server) Serve() {
	addr, _ := net.ResolveTCPAddr("tcp", ":6667")
	listener, _ := net.ListenTCP("tcp", addr)

	go server.serveUsers()

	for {
		select {
		case <-server.quit:
			return
		default:
		}

		listener.SetDeadline(time.Now().Add(time.Second))
		conn, err := listener.Accept()
		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				continue
			}

			log.Printf("Error: %v", err)
			continue
		}

		ircConn := protocol.NewIrcConnection(conn, server.incoming)
		go ircConn.Serve(server.newUsers)
	}
}

func (server *Server) serveUsers() {
	for {
		select {
		case <-server.quit:
			return
		case action := <-server.incoming:
			server.handleMessage(action)
		case action := <-server.newUsers:
			server.handleNewUser(action)
		}
	}
}
