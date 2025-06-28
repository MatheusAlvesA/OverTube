package chat_stream

import (
	"log"
	"strconv"
	"time"
)

func ConnectToYoutubeChat(channelID string) (ChatStreamCon, error) {
	con := new(YTChatStreamCon)
	con.ChannelID = channelID
	con.stream = make(chan ChatStreamMessage, ChatStreamMessageBufferSize)
	go func() {
		// Simulate receiving messages
		for i := range 5 {
			if con.stream == nil {
				return // Exit if the stream has been closed
			}
			con.stream <- ChatStreamMessage{
				Platform:  PlatformTypeYoutube,
				Name:      "User test",
				Message:   "Message " + strconv.Itoa(i),
				Timestamp: "2023-10-01T12:00:00Z",
			}
			time.Sleep(2 * time.Second)
			log.Println("Sent message:", i)
		}

		if con.stream != nil {
			close(con.stream)
			con.stream = nil // Clear the stream after closing it
		}

	}()
	return con, nil
}
