package main

import (
	"log"
	"overtube/chat_stream"
	"overtube/save_state"
	"overtube/ui"
	"overtube/web_server"
	"overtube/ws_server"
	"reflect"
)

var wsServer = ws_server.CreateServer()
var webServer = web_server.CreateServer()
var uiCommandsChan = make(chan ui.UICommand)
var appState = save_state.Read()

func main() {
	uiEventChan := make(chan ui.UIEvent)
	go ui.CreateHomeWindow(uiEventChan, uiCommandsChan, appState)
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
	var ytChatStream chat_stream.ChatStreamCon = nil
	var twChatStream chat_stream.ChatStreamCon = nil
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
			closeChatStream(ytChatStream)
			uiCommandsChan <- ui.ChannelConnectionStatusChange{
				Platform: chat_stream.PlatformTypeYoutube,
				Status:   ws_server.ChannelConnectionStarting,
			}
			var err error = nil
			ytChatStream, err = chat_stream.ConnectToYoutubeChat(v.Channel)
			if err != nil {
				log.Println("Failed to connect to YouTube chat:", v.Channel, err)
				uiCommandsChan <- ui.ChannelConnectionStatusChange{
					Platform: chat_stream.PlatformTypeYoutube,
					Status:   ws_server.ChannelConnectionStopped,
				}
			} else {
				wsServer.AddStream(ytChatStream)
				appState.YoutubeChannel = v.Channel
				save_state.Save(appState)
			}
		case ui.UIEventSetTwitchChannel:
			closeChatStream(twChatStream)
			uiCommandsChan <- ui.ChannelConnectionStatusChange{
				Platform: chat_stream.PlatformTypeTwitch,
				Status:   ws_server.ChannelConnectionStarting,
			}
			var err error = nil
			twChatStream, err = chat_stream.ConnectToTwitchChat(v.Channel)
			if err != nil {
				log.Println("Failed to connect to Twitch chat:", v.Channel, err)
				uiCommandsChan <- ui.ChannelConnectionStatusChange{
					Platform: chat_stream.PlatformTypeTwitch,
					Status:   ws_server.ChannelConnectionStopped,
				}
			} else {
				wsServer.AddStream(twChatStream)
				appState.TwitchChannel = v.Channel
				save_state.Save(appState)
			}
		case ui.UIEventRemoveYoutubeChannel:
			wsServer.RemoveAllStreamsFromPlatform(chat_stream.PlatformTypeYoutube)
			closeChatStream(ytChatStream)
		case ui.UIEventRemoveTwitchChannel:
			wsServer.RemoveAllStreamsFromPlatform(chat_stream.PlatformTypeTwitch)
			closeChatStream(twChatStream)
		case ui.UIEventExit:
			log.Println("User exited")
		default:
			log.Println("Unknown event type: ", reflect.TypeOf(event))
		}
	}

	closeChatStream(ytChatStream)
	closeChatStream(twChatStream)
}

func closeChatStream(chatStream chat_stream.ChatStreamCon) {
	if chatStream == nil || !chatStream.IsConnected() {
		return
	}
	log.Print("Closing chat stream connection...")
	chatStream.Close()
	log.Print("Chat Stream closed!")
}
