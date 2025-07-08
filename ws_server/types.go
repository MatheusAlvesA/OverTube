package ws_server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"overtube/chat_stream"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin, ok := r.Header["Origin"]
		return ok && len(origin) == 1 && origin[0] == "http://localhost:1337"
	},
}

type WSChatStreamServer struct {
	Port       uint
	srcStreams []chat_stream.ChatStreamCon
	conn       *websocket.Conn
	srv        *http.Server
}

func (s *WSChatStreamServer) Start() bool {
	s.srv = &http.Server{Addr: "localhost:" + fmt.Sprintf("%d", s.Port)}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		alreadyHandlingCon := false
		if s.conn != nil {
			s.conn.Close()
			alreadyHandlingCon = true
		}
		var err error
		s.conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[WSChatStreamServer] Fail to upgrade to WS", err)
			return
		}
		if !alreadyHandlingCon {
			go s.loopChatStreamMessages()
		}
	})

	go s.srv.ListenAndServe()

	return true
}

func (s *WSChatStreamServer) Stop() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
}

func (s *WSChatStreamServer) loopChatStreamMessages() {
	log.Println("[WSChatStreamServer] loopChatStreamMessages started")
	for {
		if s.conn == nil {
			log.Println("[WSChatStreamServer] No connection live, stop handling messages")
			return
		}
		s.handleChatStreamMessages()
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *WSChatStreamServer) handleChatStreamMessages() {
	for i, chatStream := range s.srcStreams {
		if !chatStream.IsConnected() {
			s.RemoveStream(i)
			break
		}
		select {
		case msg := <-chatStream.GetMessagesChan():
			s.conn.WriteJSON(map[string]any{
				"type":         "msg",
				"userName":     msg.Name,
				"platform":     msg.Platform,
				"timestamp":    msg.Timestamp,
				"messageParts": msg.MessageParts,
			})
		default:
			// Do nothing
		}
	}
}

func (s *WSChatStreamServer) AddStream(stream chat_stream.ChatStreamCon) {
	s.srcStreams = append(s.srcStreams, stream)
}

func (s *WSChatStreamServer) RemoveStream(i int) {
	s.srcStreams = append(s.srcStreams[:i], s.srcStreams[i+1:]...)
}
