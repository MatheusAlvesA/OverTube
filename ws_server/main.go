package ws_server

import (
	"log"
	"overtube/chat_stream"
)

func CreateServer() *WSChatStreamServer {
	server := &WSChatStreamServer{
		Port:       1336,
		srcStreams: make([]chat_stream.ChatStreamCon, 0),
	}

	log.Println("[CreateServer] Starting WS Server")
	if !server.Start() {
		return nil
	}
	log.Println("[CreateServer] WS Server started")
	return server
}
