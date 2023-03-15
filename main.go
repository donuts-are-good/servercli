package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath           = "sqlite.db"
	listenAddress    = ":8080"
	websocketUpgrade = 1024
)

func main() {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  websocketUpgrade,
		WriteBufferSize: websocketUpgrade,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server, err := NewServer(dbPath, upgrader)

	if err != nil {
		panic(fmt.Sprintf("Failed to initialize the server: %v", err))
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.HandleConnection(w, r)
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		server.HandleRegistration(w, r)
	})
	fmt.Println("Starting server on", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		panic(fmt.Sprintf("Failed to start the server: %v", err))
	}
}
