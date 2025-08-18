package ws_server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"overtube/chat_stream"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const MAX_WS_CONNS = 5

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		origin, ok := r.Header["Origin"]
		return ok && len(origin) == 1 && origin[0] == "http://localhost:1337"
	},
}

type ChannelConnectionStatus uint

const (
	ChannelConnectionStopped ChannelConnectionStatus = iota
	ChannelConnectionStarting
	ChannelConnectionRunning
)

type ChannelConnectionStatusEvent struct {
	Platform chat_stream.PlatformType
	Status   ChannelConnectionStatus
}

type WSConnection struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func (c *WSConnection) Send(data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Conn.WriteJSON(data)
}

type WSChatStreamServer struct {
	Port            uint
	srcStreams      []chat_stream.ChatStreamCon
	conns           []*WSConnection
	srv             *http.Server
	StatusEventChan chan ChannelConnectionStatusEvent
}

func (s *WSChatStreamServer) Start() bool {
	s.srv = &http.Server{Addr: "localhost:" + fmt.Sprintf("%d", s.Port)}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if len(s.srcStreams) <= 0 {
			log.Println("[WSChatStreamServer] Denying new connection, no chat stream live")
			return
		}
		if len(s.conns) >= MAX_WS_CONNS {
			log.Println("[WSChatStreamServer] Denying new connection, max connections reached")
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[WSChatStreamServer] Fail to upgrade to WS", err)
			return
		}
		newClient := &WSConnection{Conn: conn}
		s.conns = append(s.conns, newClient)
		go s.sendAllUserIdForClient(newClient)
	})

	s.StatusEventChan = make(chan ChannelConnectionStatusEvent)

	go s.srv.ListenAndServe()
	go s.loopChatStreamMessages()
	go s.clearOldConnections()

	return true
}

func (s *WSChatStreamServer) Stop() {
	s.closeAllSockets()
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
	s.srv = nil
	close(s.StatusEventChan)
	s.StatusEventChan = nil
}

func (s *WSChatStreamServer) closeAllSockets() {
	for _, conn := range s.conns {
		conn.Conn.Close()
	}
	s.conns = []*WSConnection{}
}

func (s *WSChatStreamServer) clearOldConnections() {
	for {
		if s.srv == nil {
			return
		}
		newList := []*WSConnection{}
		for _, conn := range s.conns {
			conn.Send(map[string]any{
				"type":    "cmd",
				"command": "ping",
			})
			var v map[string]any = nil
			err := conn.Conn.ReadJSON(&v)
			if err == nil && v != nil && v["command"] == "pong" {
				newList = append(newList, conn)
			} else {
				log.Println("[WSChatStreamServer] No response from client, disconnected")
				conn.Conn.Close()
			}
		}
		s.conns = newList
		time.Sleep(3 * time.Second)
	}
}

func (s *WSChatStreamServer) sendNewUserIdForAllClents(stream chat_stream.ChatStreamCon) {
	if s.srv == nil {
		return
	}
	for _, conn := range s.conns {
		s.sendNewUserId(conn, stream)
	}
}

func (s *WSChatStreamServer) sendAllUserIdForClient(conn *WSConnection) {
	time.Sleep(1 * time.Second) // Wait to guarantee client is connected
	if s.srv == nil {
		return
	}
	for _, client := range s.srcStreams {
		s.sendNewUserId(conn, client)
	}
}

func (s *WSChatStreamServer) sendNewUserId(conn *WSConnection, stream chat_stream.ChatStreamCon) {
	conn.Send(map[string]any{
		"type":     "cmd",
		"command":  "setNewUserId",
		"platform": stream.GetPlatform(),
		"id":       stream.GetUserId(),
	})
}

func (s *WSChatStreamServer) loopChatStreamMessages() {
	log.Println("[WSChatStreamServer] loopChatStreamMessages started")
	for {
		if s.srv == nil {
			log.Println("[WSChatStreamServer] No server live, stop handling messages")
			return
		}
		if len(s.srcStreams) <= 0 && len(s.conns) > 0 {
			log.Println("[WSChatStreamServer] No chat stream live, closing sockets")
			s.closeAllSockets()
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
			data := map[string]any{
				"type":         "msg",
				"userName":     msg.Name,
				"platform":     msg.Platform,
				"timestamp":    msg.Timestamp,
				"messageParts": msg.MessageParts,
				"badges":       msg.Badges,
			}
			for _, ws := range s.conns {
				ws.Send(data)
			}
		default:
			// Do nothing
		}
	}
}

func (s *WSChatStreamServer) RemoveAllStreamsFromPlatform(platform chat_stream.PlatformType) {
	newList := []chat_stream.ChatStreamCon{}
	for _, stream := range s.srcStreams {
		if stream.GetPlatform() != platform {
			newList = append(newList, stream)
		} else {
			s.StatusEventChan <- ChannelConnectionStatusEvent{
				Platform: platform,
				Status:   ChannelConnectionStopped,
			}
		}
	}
	s.srcStreams = newList
}

func (s *WSChatStreamServer) AddStream(stream chat_stream.ChatStreamCon) {
	s.srcStreams = append(s.srcStreams, stream)
	s.StatusEventChan <- ChannelConnectionStatusEvent{
		Platform: stream.GetPlatform(),
		Status:   ChannelConnectionRunning,
	}
	s.sendNewUserIdForAllClents(stream)
}

func (s *WSChatStreamServer) RefreshClients() {
	for _, conn := range s.conns {
		conn.Send(map[string]any{
			"type":    "cmd",
			"command": "refresh",
		})
	}
}

func (s *WSChatStreamServer) RemoveStream(i int) {
	s.StatusEventChan <- ChannelConnectionStatusEvent{
		Platform: s.srcStreams[i].GetPlatform(),
		Status:   ChannelConnectionStopped,
	}
	s.srcStreams = append(s.srcStreams[:i], s.srcStreams[i+1:]...)
}
