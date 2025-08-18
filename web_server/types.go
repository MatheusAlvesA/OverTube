package web_server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed www/*
var content embed.FS

type WebChatStreamServer struct {
	Port              uint
	srv               *http.Server
	selectedChatStyle *ChatStyleOption
}

func (s *WebChatStreamServer) SetSelectedChatStyle(style *ChatStyleOption) {
	s.selectedChatStyle = style
}

func (s *WebChatStreamServer) Start() bool {
	s.srv = &http.Server{Addr: "localhost:" + fmt.Sprintf("%d", s.Port)}
	staticFiles, err := fs.Sub(content, "www")
	if err != nil {
		log.Println(err)
		s.srv = nil
		return false
	}
	http.Handle("/", http.FileServer(http.FS(staticFiles)))
	http.Handle("/styles.css", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		if s.selectedChatStyle == nil {
			w.Write([]byte(""))
		} else {
			w.Write([]byte(s.selectedChatStyle.CSS))
		}
	}))
	go s.srv.ListenAndServe()
	return true
}

func (s *WebChatStreamServer) Stop() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
}

type ChatStyleOption struct {
	Id    uint
	Label string
	CSS   string
}
