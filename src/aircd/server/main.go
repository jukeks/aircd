package main

import (
    "bufio"
    "net"
    "sync"
    "log"
    "aircd/protocol"
)

var logger *log.Logger

func main() {
    log.Print("Starting server")

    listener, _ := net.Listen("tcp", ":6667")

    server := Server{[]Channel{}, []User{}, sync.Mutex{}}

    for {
        conn, _ := listener.Accept()
        go serve(&server, conn)
    }
}

func serve(server *Server, conn net.Conn) {
    defer conn.Close()

    user := NewUser(conn)
    user.handle_hostname()

    reader := bufio.NewReader(conn)

    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            log.Printf("%s %v", user.nick, err)
            server.remove_user(user)
            return
        }

        parsed := protocol.ParseMessage(message[:len(message)-2])
        server.handle_message(user, parsed)
    }
}
