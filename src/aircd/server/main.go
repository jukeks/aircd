package main

import (
    "bufio"
    "net"
    "sync"
    "log"
    "aircd/protocol"

    _ "net/http/pprof"
    "net/http"
)

var logger *log.Logger

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()

    log.Print("Starting server")

    listener, _ := net.Listen("tcp", ":6667")

    server := Server{"example.example.com", []*Channel{}, []*User{}, sync.Mutex{}}

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
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

        if err != nil || len(message) == 0 {
            log.Printf("%s %v", user.nick, err)
            server.remove_user(user)
            return
        }

        parsed := protocol.ParseMessage(message[:len(message)-2])
        server.handle_message(user, parsed)
    }
}
