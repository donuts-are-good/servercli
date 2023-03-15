package main

import "github.com/gorilla/websocket"

type User struct {
	ID        int64
	Username  string
	Password  string
	Conn      *websocket.Conn
	SendQueue chan []byte
}
