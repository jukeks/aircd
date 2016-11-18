package main

import (
	"log"

	"net/http"
	_ "net/http/pprof"

	"channeld/server"
)

var logger *log.Logger

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Print("Starting server")

	server := server.NewServer("example.example.com")
	server.Serve()
}
