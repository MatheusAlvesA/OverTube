package web_server

import (
	"log"
)

func CreateServer() *WebChatStreamServer {
	server := &WebChatStreamServer{Port: 1337}

	log.Println("[CreateServer] Starting Web Server")
	if !server.Start() {
		return nil
	}
	log.Println("[CreateServer] Web Server started")
	return server
}
