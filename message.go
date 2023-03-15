package main

type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}
