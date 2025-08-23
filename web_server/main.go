package web_server

import (
	"log"
	"overtube/save_state"
)

func CreateServer(appState *save_state.AppState) *WebChatStreamServer {
	server := &WebChatStreamServer{Port: 1337, appState: appState}

	log.Println("[CreateServer] Starting Web Server")
	if !server.Start() {
		return nil
	}
	log.Println("[CreateServer] Web Server started")
	return server
}
