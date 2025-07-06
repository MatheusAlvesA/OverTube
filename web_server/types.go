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
	Port uint
	srv  *http.Server
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
