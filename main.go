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
var uiCommandsChan = make(chan ui.UICommand)

func main() {
	uiEventChan := make(chan ui.UIEvent)
	go ui.CreateHomeWindow(uiEventChan, uiCommandsChan)
	go handleUICommands()
	orchestrateEvents(uiEventChan)
	wsServer.Stop()
	webServer.Stop()
}

func handleUICommands() {
	for {
		statusEvent, more := <-wsServer.StatusEventChan
		if !more {
			log.Println("StatusEventChan event channel closed")
			break
		}
		uiCommandsChan <- ui.ChannelConnectionStatusChange{
			Platform: statusEvent.Platform,
			Status:   statusEvent.Status,
		}
	}
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
			uiCommandsChan <- ui.ChannelConnectionStatusChange{
				Platform: chat_stream.PlatformTypeYoutube,
				Status:   ws_server.ChannelConnectionStarting,
			}
			var err error = nil
			chatStream, err = chat_stream.ConnectToYoutubeChat(v.Channel)
			if err != nil {
				log.Println("Failed to connect to YouTube chat:", v.Channel, err)
				uiCommandsChan <- ui.ChannelConnectionStatusChange{
					Platform: chat_stream.PlatformTypeYoutube,
					Status:   ws_server.ChannelConnectionStopped,
				}
			} else {
				wsServer.AddStream(chatStream)
			}
		case ui.UIEventRemoveYoutubeChannel:
			wsServer.RemoveAllStreamsFromPlatform(chat_stream.PlatformTypeYoutube)
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
