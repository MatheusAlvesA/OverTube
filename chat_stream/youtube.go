package chat_stream

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ConnectToYoutubeChat(channelID string) (ChatStreamCon, error) {
	streamUrl, err := getLiveStreamFromChannelId(channelID)
	if err != nil {
		return nil, err
	}
	log.Println("Live stream URL for channel", channelID, "is", streamUrl)
	continuationToken, err := getContinuationFromURL(streamUrl)
	if err != nil {
		return nil, err
	}
	continuationToken, err = getContinuationFromAllChat(continuationToken)
	if err != nil {
		return nil, err
	}

	return generateChatStream(channelID, continuationToken)
}

func generateChatStream(channelID string, continuationToken string) (ChatStreamCon, error) {
	log.Println("Starting YouTube chat stream for channel:", channelID)
	con := &YTChatStreamCon{
		ChannelID:         channelID,
		stream:            make(chan ChatStreamMessage, ChatStreamMessageBufferSize),
		ContinuationToken: continuationToken,
		LastStreamUpdate:  0,
	}
	go func() {
		for {
			if !con.IsConnected() {
				log.Println("Chat stream connection closed, stopping message streaming")
				return
			}
			iterateAndStreamNewMessages(con)
			time.Sleep(500 * time.Millisecond)
		}
	}()
	return con, nil
}

func iterateAndStreamNewMessages(con *YTChatStreamCon) {
	newMessages, err := iterateOnMessages(con)
	if err != nil {
		log.Println("Error iterating on messages:", err)
		con.Close()
		return
	}
	for _, msg := range newMessages {
		if !con.IsConnected() {
			log.Println("Chat stream connection closed, stopping message processing")
			return
		}
		select {
		case con.stream <- msg:
		default:
			log.Println("Chat stream buffer is full, dropping message:", msg)
		}
	}
}

func iterateOnMessages(con *YTChatStreamCon) ([]ChatStreamMessage, error) {
	response, err := getMessagesAPIResponse(con.ContinuationToken)
	if err != nil {
		return nil, err
	}

	newContinuationToken, err := getContinuationFromAPIResponse(response)
	if err != nil {
		return nil, err
	}
	con.ContinuationToken = newContinuationToken

	messages := make([]ChatStreamMessage, 0)
	messagesData, ok := GetDeepMapValue(response, []any{
		"continuationContents",
		"liveChatContinuation",
		"actions",
	}, false)
	if !ok {
		return messages, nil
	}
	actions, ok := messagesData.([]any)
	if !ok {
		return nil, &CustomError{message: "Actions data is not in expected format"}
	}

	for _, action := range actions {
		item, ok := GetDeepMapValue(action.(map[string]any), []any{
			"addChatItemAction",
			"item",
			"liveChatTextMessageRenderer",
		}, true)
		if !ok {
			continue
		}
		message, err := getMessageFromChatItem(item.(map[string]any), con.LastStreamUpdate)
		if err != nil {
			log.Println("Error getting message from chat item:", err)
			continue
		}
		if message == nil {
			continue
		}
		messages = append(messages, *message)
	}
	if len(messages) > 0 {
		con.LastStreamUpdate = messages[len(messages)-1].Timestamp
	}

	return messages, nil
}

func getMessageFromChatItem(item map[string]any, lastTimeUpdate int64) (*ChatStreamMessage, error) {
	timestamp, ok := GetDeepMapValue(item, []any{
		"timestampUsec",
	}, false)
	if !ok {
		return nil, &CustomError{message: "Timestamp not found in chat item"}
	}
	timestampInt, err := strconv.ParseInt(timestamp.(string), 10, 64)
	if err != nil {
		return nil, err
	}
	timestampInt = timestampInt / 1000

	if timestampInt <= lastTimeUpdate {
		return nil, nil // Skip messages that are older than the last update
	}

	name, ok := GetDeepMapValue(item, []any{
		"authorName",
		"simpleText",
	}, false)
	if !ok {
		return nil, &CustomError{message: "Author name not found in chat item"}
	}

	messagesText, ok := GetDeepMapValue(item, []any{
		"message",
		"runs",
	}, false)
	if !ok {
		return nil, &CustomError{message: "Message text not found in chat item"}
	}
	var messageParts []ChatStreamMessagePart
	for _, messageEntry := range messagesText.([]any) {
		message, err := getMessagePartFromChatItemEntry(messageEntry.(map[string]any))
		if err != nil {
			log.Println("Error getting message part from chat item entry:", err)
			continue
		}
		messageParts = append(messageParts, message)
	}

	if len(messageParts) == 0 {
		return nil, &CustomError{message: "No valid message parts found in chat item"}
	}

	return &ChatStreamMessage{
		Platform:     PlatformTypeYoutube,
		Name:         name.(string),
		MessageParts: messageParts,
		Timestamp:    timestampInt,
	}, nil
}

func getMessagePartFromChatItemEntry(messageEntry map[string]any) (ChatStreamMessagePart, error) {
	text, ok := GetDeepMapValue(messageEntry, []any{
		"text",
	}, true)
	if ok {
		return ChatStreamMessagePart{
			PartType: ChatStreamMessagePartTypeText,
			Text:     text.(string),
		}, nil
	}
	message := ChatStreamMessagePart{
		PartType: ChatStreamMessagePartTypeEmote,
	}
	emojiText, ok := GetDeepMapValue(messageEntry, []any{
		"emoji",
		"emojiId",
	}, true)
	if ok {
		message.Text = emojiText.(string)
	}
	emojiImg, ok := GetDeepMapValue(messageEntry, []any{
		"emoji",
		"image",
		"thumbnails",
		0,
		"url",
	}, true)
	if !ok {
		return message, &CustomError{message: "Emote image URL not found in message entry"}
	}
	message.EmoteImgUrl = emojiImg.(string)
	emojiName, ok := GetDeepMapValue(messageEntry, []any{
		"emoji",
		"shortcuts",
		0,
	}, true)
	if !ok {
		return message, &CustomError{message: "Emote name not found in message entry"}
	}
	message.EmoteName = emojiName.(string)

	if message.Text == "" {
		message.Text = message.EmoteName // Fallback to emote name if no text is
	}

	return message, nil
}

func getMessagesAPIResponse(continuationToken string) (map[string]any, error) {
	reqData := map[string]any{
		"continuation": continuationToken,
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "pt",
				"gl":            "BR",
				"clientName":    "WEB",
				"clientVersion": "2.20250626.01.00",
			},
		},
	}
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(
		"https://www.youtube.com/youtubei/v1/live_chat/get_live_chat?prettyPrint=false",
		"application/json",
		strings.NewReader(string(jsonBytes)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, http.ErrNotSupported
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func getContinuationFromAPIResponse(response map[string]any) (string, error) {
	ifreeChatContinuationToken, ok := GetDeepMapValue(response, []any{
		"continuationContents",
		"liveChatContinuation",
		"header",
		"liveChatHeaderRenderer",
		"viewSelector",
		"sortFilterSubMenuRenderer",
		"subMenuItems",
		-1,
		"continuation",
		"reloadContinuationData",
		"continuation",
	}, false)
	if !ok {
		return "", &CustomError{message: "Continuation token not found in response"}
	}

	return ifreeChatContinuationToken.(string), nil
}

func getContinuationFromURL(streamUrl string) (string, error) {
	resp, err := http.Get(streamUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", http.ErrNotSupported
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	index := strings.Index(string(body), "{\"liveChatRenderer\":{\"continuations\":[{\"reloadContinuationData\":{\"continuation\":\"")
	if index < 0 {
		return "", &CustomError{message: "Continuation not found in response"}
	}
	index += 82 // Length of the string before the continuation value
	endIndex := strings.Index(string(body)[index:], "\"")
	if endIndex < 0 {
		return "", &CustomError{message: "End of continuation not found"}
	}

	return string(body)[index-1 : index+endIndex], nil
}

func getContinuationFromAllChat(continuationToken string) (string, error) {
	parsed, err := getMessagesAPIResponse(continuationToken)
	if err != nil {
		return "", err
	}

	return getContinuationFromAPIResponse(parsed)
}

func getLiveStreamFromChannelId(channelID string) (string, error) {
	resp, err := http.Get("https://www.youtube.com/@" + channelID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &CustomError{message: "Failed to fetch channel page, status code: " + strconv.Itoa(resp.StatusCode)}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	indexOffset := 71
	index := strings.Index(string(body), "channelFeaturedContentRenderer\":{\"items\":[{\"videoRenderer\":{\"videoId\":\"")
	if index < 0 {
		index = strings.Index(string(body), "\"channelVideoPlayerRenderer\":{\"videoId\":\"")
		indexOffset = 41
		if index < 0 {
			return "", &CustomError{message: "Live stream data not found in response"}
		}
	}
	index += indexOffset // Length of the string before the JSON data
	endIndex := strings.Index(string(body)[index:], "\"")
	if endIndex < 0 {
		return "", &CustomError{message: "End of live stream code data not found"}
	}

	return ("https://www.youtube.com/watch?v=" + string(body)[index:index+endIndex]), nil
}
