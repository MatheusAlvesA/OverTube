package chat_stream

import (
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

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

	fillBadgesDatabase(con)

	return nil
}

func fillBadgesDatabase(con *TWChatStreamCon) {
	var badges map[string]ChatUserBadge = map[string]ChatUserBadge{}

	badges["moderator/1"] = ChatUserBadge{
		Name:   "Moderator",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/3267646d-33f0-4b17-b3df-f923a41db1d0/3",
		Type:   "moderator",
	}
	badges["staff/1"] = ChatUserBadge{
		Name:   "Twitch Staff",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/d97c37bd-a6f5-4c38-8f57-4e4bef88af34/3",
		Type:   "staff",
	}
	badges["vip/1"] = ChatUserBadge{
		Name:   "VIP",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/b817aba4-fad8-49e2-b88a-7cc744dfa6ec/3",
		Type:   "vip",
	}
	badges["partner/1"] = ChatUserBadge{
		Name:   "Verified",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/d12a2e27-16f6-41d0-ab77-b780518f00a3/3",
		Type:   "partner",
	}
	badges["founder/0"] = ChatUserBadge{
		Name:   "Founder",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/511b78a9-ab37-472f-9569-457753bbe7d3/3",
		Type:   "founder",
	}
	badges["premium/1"] = ChatUserBadge{
		Name:   "Prime",
		ImgSrc: "https://static-cdn.jtvnw.net/badges/v1/bbbe0db0-a598-423e-86d0-f9fb98ca1933/3",
		Type:   "prime",
	}

	con.badgesDB = badges
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
		parsed, err := parseTwMessage(con, string(message))
		if err != nil {
			continue
		}
		con.stream <- *parsed
	}
}

func parseTwMessage(con *TWChatStreamCon, message string) (*ChatStreamMessage, error) {
	data, err := explodeTwMessage(message)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.Atoi(data["tmi-sent-ts"])
	if err != nil {
		timestamp = int(time.Now().Unix())
	}

	messageParts := []ChatStreamMessagePart{}
	messageParts = append(messageParts, ChatStreamMessagePart{
		PartType: ChatStreamMessagePartTypeText,
		Text:     data["message"],
	})

	res := &ChatStreamMessage{
		Platform:     PlatformTypeTwitch,
		Name:         data["display-name"],
		MessageParts: messageParts,
		Timestamp:    int64(timestamp / 1000),
		Badges:       parseBadges(con, data),
	}

	return res, nil
}

func parseBadges(con *TWChatStreamCon, metaData map[string]string) []ChatUserBadge {
	badges := []ChatUserBadge{}
	rawList := strings.Split(metaData["badges"], ",")
	if len(rawList) == 0 {
		return badges
	}
	for _, pair := range rawList {
		if pair == "" {
			continue
		}
		badgeEntry, ok := con.badgesDB[pair]
		if !ok {
			continue
		}
		badges = append(badges, badgeEntry)
	}
	return badges
}

func explodeTwMessage(message string) (map[string]string, error) {
	parts := strings.Split(message, "PRIVMSG")
	if len(parts) < 2 {
		return nil, &CustomError{"PRIVMSG not found in message"}
	}
	metaData := strings.Split(parts[0], ";")
	if len(parts) < 2 {
		return nil, &CustomError{"Invalid message format, no ';' found"}
	}
	res := make(map[string]string)
	res["message"] = strings.Split(parts[1], ":")[1]

	for _, line := range metaData {
		currentEntry := strings.Split(line, "=")
		if len(currentEntry) != 2 {
			return nil, &CustomError{"Invalid metadata format, expected key=value"}
		}
		key := strings.TrimSpace(currentEntry[0])
		value := strings.TrimSpace(currentEntry[1])
		res[key] = value
	}

	return res, nil
}
