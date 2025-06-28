package main

import (
	"overtube/ui"
)

func main() {
	var uiDone = make(chan bool)
	go ui.CreateHomeWindow(uiDone)
	<-uiDone
}
