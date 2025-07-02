package ws_server

import "overtube/chat_stream"

func CreateServer() *WSChatStreamServer {
	server := &WSChatStreamServer{
		Port:       1336,
		srcStreams: make([]chat_stream.ChatStreamCon, 0),
	}

	if !server.Start() {
		return nil
	}
	return server
}
