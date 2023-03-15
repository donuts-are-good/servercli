package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	user, err := s.Database.GetUser(username)
	if err != nil {
		log.Printf("Error: %v", err)
		_ = conn.WriteMessage(websocket.CloseMessage, []byte("Invalid username or password"))
		conn.Close()
		return
	}

	if user.Password != password {
		log.Printf("Error: %v", err)
		_ = conn.WriteMessage(websocket.CloseMessage, []byte("Invalid username or password"))
		conn.Close()
		return
	}

	user.Conn = conn
	user.SendQueue = make(chan []byte, 256)
	s.AddUser(user)
	defer func() {
		s.RemoveUser(username)
		conn.Close()
	}()

	go s.WriteMessages(user)
	s.ReadMessages(user)
}

func (s *Server) ReadMessages(user *User) {
	for {
		_, msg, err := user.Conn.ReadMessage()
		if err != nil {
			return
		}
		log.Printf("Received message from %s: %s", user.Username, msg) // Add this log statement

		message, err := parseMessage(msg)
		if err != nil {
			continue
		}

		if err := s.HandleMessage(user, message); err != nil {
			continue
		}
	}
}

func parseMessage(msg []byte) (*Message, error) {
	var message Message
	err := json.Unmarshal(msg, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

