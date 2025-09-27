package chat_stream

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
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

	fillCustomBadgesDatabase(con)
}

func fillCustomBadgesDatabase(con *TWChatStreamCon) {
	reqData := map[string]any{
		"operationName": "ChatList_Badges",
		"variables": map[string]any{
			"channelLogin": con.ChannelID,
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "838a7e0b47c09cac05f93ff081a9ff4f876b68f7624f0fc465fe30031e372fc2",
			},
		},
	}
	jsonBytes, err := json.Marshal([]map[string]any{reqData})
	if err != nil {
		log.Println("Error marshaling JSON for Twitch badges:", err)
		return
	}

	req, _ := http.NewRequest(http.MethodPost, "https://gql.twitch.tv/gql", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("client-id", "kimne78kx3ncx6brgo4mv6wki5h1ko")
	req.Body = io.NopCloser(strings.NewReader(string(jsonBytes)))

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)

	if err != nil {
		log.Println("Fail to get custom badges from channel", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("Fail to get custom badges from channel, status: ", resp.StatusCode)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error parsing JSON from Twitch badges:", err)
		return
	}

	var data []map[string]any
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Println("Error unmarshaling JSON from Twitch badges:", err)
		return
	}

	userIdStr, ok := GetDeepMapValue(data[0], []any{"data", "user", "id"}, false)
	if !ok {
		log.Println("Error getting userId from Twitch response")
	} else {
		con.UserID = userIdStr.(string)
	}

	badgesMap, ok := GetDeepMapValue(data[0], []any{"data", "user", "broadcastBadges"}, false)
	if !ok {
		log.Println("Error getting custom badges from Twitch response")
		return
	}

	for _, badge := range badgesMap.([]any) {
		badgeData, ok := badge.(map[string]any)
		if !ok {
			log.Println("Error parsing badge data from Twitch response", badge)
			continue
		}
		con.badgesDB[badgeData["setID"].(string)+"/"+badgeData["version"].(string)] = ChatUserBadge{
			Name:   badgeData["title"].(string),
			ImgSrc: badgeData["image4x"].(string),
			Type:   "custom",
		}
	}
}

func iterateOnTwMessages(con *TWChatStreamCon) {
	pingTimerval := time.NewTicker(30 * time.Second)
	defer pingTimerval.Stop()
	for {
		if !con.IsConnected() {
			con.Close()
			return
		}
		select {
		case <-pingTimerval.C:
			con.ws.WriteMessage(websocket.TextMessage, []byte("PING"))
		default:
			// Continue to read messages
		}

		_, message, err := con.ws.ReadMessage()
		if err != nil {
			log.Println(err)
			con.Close()
			return
		}
		parsed, err := parseTwMessage(con, string(message))
		if err != nil || parsed == nil {
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

	parts, err := parseMessageParts(data["message"], data["emotes"])
	if err != nil {
		log.Println("Error parsing message parts:", err)
		log.Println("Raw Message:", message)
		log.Println("Emotes:", data["emotes"])
		log.Println("Message:", data["message"])
		return nil, err
	}

	res := &ChatStreamMessage{
		Platform:     PlatformTypeTwitch,
		Name:         data["display-name"],
		MessageParts: parts,
		Timestamp:    int64(timestamp / 1000),
		Badges:       parseBadges(con, data),
	}

	return res, nil
}

func parseMessageParts(message string, emotes string) ([]ChatStreamMessagePart, error) {
	parts := []ChatStreamMessagePart{}
	if message == "" {
		return parts, nil
	}

	if emotes == "" {
		parts = append(parts, ChatStreamMessagePart{
			PartType: ChatStreamMessagePartTypeText,
			Text:     message,
		})
		return parts, nil
	}

	emoteRawList := strings.Split(emotes, "/")
	emoteParsedList := []map[string]any{}
	for _, emoteRaw := range emoteRawList {
		if emoteRaw == "" {
			log.Println("Empty emote found in Twitch message, skipping")
			continue
		}
		emoteParts := strings.Split(emoteRaw, ":")
		if len(emoteParts) != 2 {
			log.Println("Invalid emote format in Twitch message, expected 'id:start-end', got:", emotes)
			continue
		}
		emoteUrl := "https://static-cdn.jtvnw.net/emoticons/v2/" + emoteParts[0] + "/default/dark/2.0"

		emoteRanges := strings.SplitSeq(emoteParts[1], ",")
		for emoteRange := range emoteRanges {
			emotePositions := strings.Split(emoteRange, "-")
			if len(emotePositions) != 2 {
				log.Println("Invalid emote positions in Twitch message, expected 'start-end', got:", emoteRaw)
				continue
			}
			start, err := strconv.Atoi(emotePositions[0])
			if err != nil {
				log.Println("Invalid start position in Twitch emote, skipping:", emoteRaw, err)
				continue
			}
			end, err := strconv.Atoi(emotePositions[1])
			if err != nil {
				log.Println("Invalid end position in Twitch emote, skipping:", emoteRaw, err)
				continue
			}

			if start > len(message) {
				return nil, &CustomError{message: "Emote start position(" + strconv.Itoa(start) + ") is greater than message length(" + strconv.Itoa(len(message)) + ")"}
			}

			emoteParsedList = append(emoteParsedList, map[string]any{
				"start": start,
				"end":   end,
				"emote": ChatStreamMessagePart{
					PartType:    ChatStreamMessagePartTypeEmote,
					EmoteImgUrl: emoteUrl,
					EmoteName:   message[start:int(math.Min(float64(end+1), float64(len(message))))],
				},
			})
		}
	}

	sort.Slice(emoteParsedList, func(i, j int) bool {
		return emoteParsedList[i]["start"].(int) < emoteParsedList[j]["start"].(int)
	})

	cursor := 0
	for _, emoteData := range emoteParsedList {
		start := emoteData["start"].(int)
		end := emoteData["end"].(int)

		if start < cursor {
			return nil, &CustomError{message: "Emote start position is less than cursor:" + strconv.Itoa(start) + " < " + strconv.Itoa(cursor)}
		}
		if start > cursor {
			parts = append(parts, ChatStreamMessagePart{
				PartType: ChatStreamMessagePartTypeText,
				Text:     message[cursor:start],
			})
		}
		parts = append(parts, emoteData["emote"].(ChatStreamMessagePart))
		cursor = end + 1
	}

	if cursor < len(message) {
		parts = append(parts, ChatStreamMessagePart{
			PartType: ChatStreamMessagePartTypeText,
			Text:     message[cursor:],
		})
	}

	return parts, nil
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
	res["message"] = strings.Join(strings.Split(parts[1], ":")[1:], ":")

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
