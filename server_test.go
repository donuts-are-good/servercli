package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

// TestServerCreation tests the creation of a new server.
func TestServerCreation(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}

	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.Database == nil {
		t.Fatal("Server database should not be nil")
	}

	if server.Users == nil {
		t.Fatal("Server users map should not be nil")
	}
}

// TestHandleRegistration tests the registration handler.
func TestHandleRegistration(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}

	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req, err := http.NewRequest("GET", "/register?username=testUser&password=testPassword", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleRegistration)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HandleRegistration handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "User created successfully"
	if rr.Body.String() != expected {
		t.Errorf("HandleRegistration handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestAddUser(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}

	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	user := &User{
		Username:  "testUser",
		SendQueue: make(chan []byte),
	}

	server.AddUser(user)

	if _, ok := server.Users["testUser"]; !ok {
		t.Error("User not added to the server")
	}
}

func TestRemoveUser(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}

	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	user := &User{
		Username:  "testUser",
		SendQueue: make(chan []byte),
	}

	server.AddUser(user)
	server.RemoveUser("testUser")

	if _, ok := server.Users["testUser"]; ok {
		t.Error("User not removed from the server")
	}
}
func TestHandleAddFriend(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.Database.CreateUser("userA", "password")
	server.Database.CreateUser("userB", "password")
	userA := &User{Username: "userA"}
	userB := &User{
		Username:  "userB",
		SendQueue: make(chan []byte, 1),
	}
	server.AddUser(userB)
	message := &Message{
		Type: "add_friend",
		Data: map[string]interface{}{
			"username": "userB",
		},
	}
	err = server.handleAddFriend(userA, message)
	if err != nil {
		t.Fatalf("handleAddFriend failed: %v", err)
	}
	row := server.Database.db.QueryRow("SELECT COUNT(*) FROM friends WHERE (user_id = (SELECT id FROM users WHERE username = ?) AND friend_id = (SELECT id FROM users WHERE username = ?)) OR (user_id = (SELECT id FROM users WHERE username = ?) AND friend_id = (SELECT id FROM users WHERE username = ?))", "userA", "userB", "userB", "userA")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatalf("Database error: %v", err)
	}
	if count != 1 {
		t.Error("Users should be friends after add_friend message")
	}
	select {
	case msg := <-userB.SendQueue:
		var receivedMessage Message
		err = json.Unmarshal(msg, &receivedMessage)
		if err != nil {
			t.Fatalf("JSON unmarshal error: %v", err)
		}
		if receivedMessage.Type != "friend_request" || receivedMessage.Data["username"] != "userA" {
			t.Error("Friend request message not sent to userB")
		}
	default:
		t.Error("Friend request message not sent to userB")
	}
}

func TestHandleRemoveFriend(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.Database.CreateUser("userA", "password")
	server.Database.CreateUser("userB", "password")
	server.Database.AddFriend("userA", "userB")
	userA := &User{Username: "userA"}
	message := &Message{
		Type: "remove_friend",
		Data: map[string]interface{}{
			"username": "userB",
		},
	}
	err = server.handleRemoveFriend(userA, message)
	if err != nil {
		t.Fatalf("handleRemoveFriend failed: %v", err)
	}
	row := server.Database.db.QueryRow("SELECT COUNT(*) FROM friends WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", "userA", "userB", "userB", "userA")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatalf("Database error: %v", err)
	}
	if count != 0 {
		t.Error("Users should not be friends after remove_friend message")
	}
}

func TestHandleMessage(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.Database.CreateUser("userA", "password")
	server.Database.CreateUser("userB", "password")
	userA := &User{Username: "userA"}
	userB := &User{
		Username:  "userB",
		SendQueue: make(chan []byte, 1),
	}
	server.AddUser(userB)
	message := &Message{
		Type: "send_message",
		Data: map[string]interface{}{
			"username": "userB",
			"content":  "Hello, userB!",
		},
	}
	err = server.handleMessage(userA, message)
	if err != nil {
		t.Fatalf("handleMessage failed: %v", err)
	}
	select {
	case msg := <-userB.SendQueue:
		var receivedMessage Message
		err = json.Unmarshal(msg, &receivedMessage)
		if err != nil {
			t.Fatalf("JSON unmarshal error: %v", err)
		}
		if receivedMessage.Type != "message" || receivedMessage.Data["username"] != "userA" || receivedMessage.Data["content"] != "Hello, userB!" {
			t.Error("Message not sent or received correctly")
		}
	default:
		t.Error("Message not sent or received")
	}
}

func TestHandleFriendResponseAccepted(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.Database.CreateUser("userA", "password")
	server.Database.CreateUser("userB", "password")
	userA := &User{Username: "userA"}
	userB := &User{
		Username:  "userB",
		SendQueue: make(chan []byte, 1),
	}
	server.AddUser(userA)
	server.AddUser(userB)
	message := &Message{
		Type: "friend_response",
		Data: map[string]interface{}{
			"username": "userA",
			"accepted": true,
		},
	}
	err = server.handleFriendResponse(userB, message)
	if err != nil {
		t.Fatalf("handleFriendResponse failed: %v", err)
	}
	row := server.Database.db.QueryRow("SELECT COUNT(*) FROM friends WHERE (user_id = (SELECT id FROM users WHERE username = ?) AND friend_id = (SELECT id FROM users WHERE username = ?)) OR (user_id = (SELECT id FROM users WHERE username = ?) AND friend_id = (SELECT id FROM users WHERE username = ?))", "userA", "userB", "userB", "userA")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatalf("Database error: %v", err)
	}
	if count != 1 {
		t.Error("Users should be friends after accepted friend_response")
	}
}

func TestSendToUser(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	userA := &User{
		Username:  "userA",
		SendQueue: make(chan []byte, 1),
	}
	userB := &User{
		Username:  "userB",
		SendQueue: make(chan []byte, 1),
	}
	server.AddUser(userA)
	server.AddUser(userB)
	message := []byte("Test message")
	server.SendToUser("userB", message)
	receivedMessage := <-userB.SendQueue
	if !bytes.Equal(receivedMessage, message) {
		t.Error("Sent message does not match received message")
	}
}

func TestHandleRegistrationCreateUserError(t *testing.T) {
	dbPath := ":memory:"
	upgrader := websocket.Upgrader{}
	server, err := NewServer(dbPath, upgrader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(server.HandleRegistration))
	defer ts.Close()
	server.Database.CreateUser("userA", "password")
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?username=userA&password=password", ts.URL), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status code %d, got %d", http.StatusConflict, resp.StatusCode)
	}
}
