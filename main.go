package main

import (
	"log"
	"overtube/ui"
	"reflect"
)

func main() {
	uiEventChan := make(chan ui.UIEvent)
	go ui.CreateHomeWindow(uiEventChan)
	orchestrateEvents(uiEventChan)
}

func orchestrateEvents(uiEventChan chan ui.UIEvent) {
	for {
		event, more := <-uiEventChan
		if !more {
			log.Println("UI event channel closed")
			return
		}
		if event.GetError() != nil {
			log.Fatalln(event.GetError())
		}

		switch v := event.(type) {
		case ui.UIEventSetYoutubeChannel:
			log.Println("Set channel: ", v.Channel)
		case ui.UIEventExit:
			log.Println("User exited")
		default:
			log.Fatalln("Unknown event type:", reflect.TypeOf(event))
		}
	}
}
