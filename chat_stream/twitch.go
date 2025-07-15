package chat_stream

import (
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

func ConnectToTwitchChat(channelID string) (ChatStreamCon, error) {
	return generateTwChatStream(channelID)
}

func generateTwChatStream(channelID string) (ChatStreamCon, error) {
	log.Println("Starting Twitch chat stream for channel:", channelID)
	con := &TWChatStreamCon{
		ChannelID: channelID,
		stream:    make(chan ChatStreamMessage, ChatStreamMessageBufferSize),
	}

	u := url.URL{Scheme: "wss", Host: "irc-ws.chat.twitch.tv", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	con.ws = c

	err = initTwitchChatStream(con)
	if err != nil {
		log.Println("Error initializing Twitch chat stream:", err)
		con.Close()
		return nil, err
	}

	go iterateOnTwMessages(con)

	return con, nil
}

func initTwitchChatStream(con *TWChatStreamCon) error {
	err := con.ws.WriteMessage(websocket.TextMessage, []byte("CAP REQ :twitch.tv/tags twitch.tv/commands"))
	if err != nil {
		log.Println(err)
		return err
	}

	err = con.ws.WriteMessage(websocket.TextMessage, []byte("PASS SCHMOOPIIE"))
	if err != nil {
		log.Println(err)
		return err
	}

	err = con.ws.WriteMessage(websocket.TextMessage, []byte("NICK justinfan12345"))
	if err != nil {
		log.Println(err)
		return err
	}

	err = con.ws.WriteMessage(websocket.TextMessage, []byte("USER justinfan12345 8 * :justinfan12345"))
	if err != nil {
		log.Println(err)
		return err
	}

	err = con.ws.WriteMessage(websocket.TextMessage, []byte("JOIN #"+con.ChannelID))
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func iterateOnTwMessages(con *TWChatStreamCon) {
	for {
		if !con.IsConnected() {
			con.Close()
			return
		}
		_, message, err := con.ws.ReadMessage()
		if err != nil {
			log.Println(err)
			con.Close()
			return
		}
		log.Println(string(message))
	}
}
