package main

import (
	"log"
	"overtube/chat_stream"
	"overtube/ui"
	"overtube/web_server"
	"overtube/ws_server"
	"reflect"
)

var wsServer = ws_server.CreateServer()
var webServer = web_server.CreateServer()

func main() {
	uiEventChan := make(chan ui.UIEvent)
	go ui.CreateHomeWindow(uiEventChan)
	orchestrateEvents(uiEventChan)
	wsServer.Stop()
	webServer.Stop()
}

func orchestrateEvents(uiEventChan chan ui.UIEvent) {
	var chatStream chat_stream.ChatStreamCon = nil
	for {
		event, more := <-uiEventChan
		if !more {
			log.Println("UI event channel closed")
			break
		}
		if event.GetError() != nil {
			log.Fatalln(event.GetError())
		}

		switch v := event.(type) {
		case ui.UIEventSetYoutubeChannel:
			closeChatStream(chatStream)
			var err error = nil
			chatStream, err = chat_stream.ConnectToYoutubeChat(v.Channel)
			if err != nil {
				log.Println("Failed to connect to YouTube chat: ", err)
			} else {
				wsServer.AddStream(chatStream)
			}
		case ui.UIEventExit:
			log.Println("User exited")
		default:
			log.Println("Unknown event type: ", reflect.TypeOf(event))
		}
	}

	closeChatStream(chatStream)
}

func closeChatStream(chatStream chat_stream.ChatStreamCon) {
	if chatStream == nil || !chatStream.IsConnected() {
		return
	}
	log.Print("Closing chat stream connection...")
	chatStream.Close()
	log.Print("Chat Stream closed!")
}
