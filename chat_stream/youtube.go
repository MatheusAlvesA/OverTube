package chat_stream

import (
	"encoding/json"
	"io"
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
				Name:      continuationToken,
				Message:   "Message " + strconv.Itoa(i),
				Timestamp: "2023-10-01T12:00:00Z",
			}
			time.Sleep(2 * time.Second)
		}

		if con.stream != nil {
			close(con.stream)
			con.stream = nil // Clear the stream after closing it
		}

	}()
	return con, nil
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
		return "", err
	}
	resp, err := http.Post(
		"https://www.youtube.com/youtubei/v1/live_chat/get_live_chat?prettyPrint=false",
		"application/json",
		strings.NewReader(string(jsonBytes)),
	)
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

	var parsed map[string]any
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return "", err
	}

	freeChatContinuationToken, ok := GetDeepMapValue(parsed, []any{
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
	})
	if !ok {
		return "", &CustomError{message: "Continuation token not found in response"}
	}

	return freeChatContinuationToken.(string), nil
}
