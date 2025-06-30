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
	continuationToken, err := getContinuationFromURL(channelID)
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
	con := new(YTChatStreamCon)
	con.ChannelID = channelID
	con.stream = make(chan ChatStreamMessage, ChatStreamMessageBufferSize)
	con.ContinuationToken = continuationToken
	con.LastStreamUpdate = 0
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
	var messageText string = ""
	for _, messageEntry := range messagesText.([]any) {
		text, ok := GetDeepMapValue(messageEntry.(map[string]any), []any{
			"text",
		}, false)
		if !ok {
			continue
		}
		messageText = messageText + text.(string)
	}

	return &ChatStreamMessage{
		Platform:  PlatformTypeYoutube,
		Name:      name.(string),
		Message:   messageText,
		Timestamp: timestampInt,
	}, nil
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

func getContinuationFromURL(url string) (string, error) {
	resp, err := http.Get(url)
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
