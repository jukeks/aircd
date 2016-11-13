package main

import (
	"log"

	"net/http"
	_ "net/http/pprof"
)

var logger *log.Logger

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Print("Starting server")

	server := NewServer("example.example.com")
	server.Serve()
}
