package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	Users    map[string]*User
	Database *Database
	Upgrader websocket.Upgrader
}

func NewServer(dbPath string, upgrader websocket.Upgrader) (*Server, error) {
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	return &Server{
		Users:    make(map[string]*User),
		Database: db,
		Upgrader: upgrader,
	}, nil
}

func (s *Server) AddUser(user *User) {
	s.Users[user.Username] = user
}

func (s *Server) RemoveUser(username string) {
	delete(s.Users, username)
}

func (s *Server) HandleMessage(user *User, message *Message) error {
	switch message.Type {
	case "add_friend":
		return s.handleAddFriend(user, message)
	case "remove_friend":
		return s.handleRemoveFriend(user, message)
	case "message":
		return s.handleMessage(user, message)
	case "friend_response":
		return s.handleFriendResponse(user, message)
	default:
		return fmt.Errorf("unknown message type: %s", message.Type)
	}
}

func (s *Server) handleAddFriend(user *User, message *Message) error {
	friendUsername, _ := message.Data["username"].(string)
	err := s.Database.AddFriend(user.Username, friendUsername)
	if err != nil {
		return err
	}
	if _, ok := s.Users[friendUsername]; ok {
		friendReq := &Message{
			Type: "friend_request",
			Data: map[string]interface{}{"username": user.Username},
		}
		friendReqJSON, _ := json.Marshal(friendReq)
		s.SendToUser(friendUsername, friendReqJSON)
	}
	return nil
}

func (s *Server) handleRemoveFriend(user *User, message *Message) error {
	friendUsername, _ := message.Data["username"].(string)
	return s.Database.RemoveFriend(user.Username, friendUsername)
}

func (s *Server) handleMessage(user *User, message *Message) error {
	recipient, _ := message.Data["username"].(string)
	content, _ := message.Data["content"].(string)
	chatMsg := &Message{
		Type: "message",
		Data: map[string]interface{}{
			"username": user.Username,
			"content":  content,
		},
	}
	chatMsgJSON, _ := json.Marshal(chatMsg)
	s.SendToUser(recipient, chatMsgJSON)
	return nil
}

func (s *Server) handleFriendResponse(user *User, message *Message) error {
	friendUsername, _ := message.Data["username"].(string)
	accepted, _ := message.Data["accepted"].(bool)
	if accepted {
		return s.Database.AddFriend(user.Username, friendUsername)
	}
	return s.Database.RemoveFriend(user.Username, friendUsername)
}

func (s *Server) WriteMessages(user *User) {
	for message := range user.SendQueue {
		err := user.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing message for user %s: %v", user.Username, err)
			user.Conn.Close()
			s.RemoveUser(user.Username)
			return
		}
	}
}

func (s *Server) SendToUser(username string, message []byte) {
	if user, ok := s.Users[username]; ok {
		user.SendQueue <- message
	}
}

func (s *Server) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	log.Printf("Registering user: %s", username)

	_, err := s.Database.GetUser(username)
	if err == nil {
		// User already exists
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Username already exists"))
		return
	}

	err = s.Database.CreateUser(username, password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create user"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User created successfully"))
}
