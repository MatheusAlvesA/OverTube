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
	userId, err := getUserIdFromURL(streamUrl)
	if err != nil {
		return nil, err
	}

	return generateChatStream(channelID, userId, continuationToken)
}

func generateChatStream(channelID string, userId string, continuationToken string) (ChatStreamCon, error) {
	log.Println("Starting YouTube chat stream for channel:", channelID, "with user ID:", userId)
	con := &YTChatStreamCon{
		ChannelID:         channelID,
		UserID:            userId,
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
			time.Sleep(100 * time.Millisecond)
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
			log.Println(err)
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
		Badges:       getBadgesFromChatItem(item),
	}, nil
}

func getBadgesFromChatItem(item map[string]any) []ChatUserBadge {
	badgesData, ok := GetDeepMapValue(item, []any{
		"authorBadges",
	}, true)
	if !ok {
		return []ChatUserBadge{}
	}
	badges, ok := badgesData.([]any)
	if !ok {
		log.Println("Badges data is not in expected format, returning empty badges")
		return []ChatUserBadge{}
	}

	var chatBadges []ChatUserBadge
	for _, badge := range badges {
		badgeMap := badge.(map[string]any)
		name, ok := GetDeepMapValue(badgeMap, []any{
			"liveChatAuthorBadgeRenderer",
			"tooltip",
		}, false)
		if !ok {
			continue
		}

		badgeType, ok := GetDeepMapValue(badgeMap, []any{
			"liveChatAuthorBadgeRenderer",
			"icon",
			"iconType",
		}, true)
		if !ok {
			badgeType = "CUSTOM"
		}

		imgSrc := "https://upload.wikimedia.org/wikipedia/commons/thumb/d/d9/Icon-round-Question_mark.svg/240px-Icon-round-Question_mark.svg.png"
		if badgeType == "VERIFIED" {
			imgSrc = "https://static-cdn.jtvnw.net/badges/v1/d12a2e27-16f6-41d0-ab77-b780518f00a3/3"
		}
		if badgeType == "MODERATOR" {
			imgSrc = "https://static-cdn.jtvnw.net/badges/v1/3267646d-33f0-4b17-b3df-f923a41db1d0/3"
		}
		if badgeType == "OWNER" {
			imgSrc = "https://static-cdn.jtvnw.net/badges/v1/5527c58c-fb7d-422d-b71b-f309dcb85cc1/3"
		}

		customUrl, ok := GetDeepMapValue(badgeMap, []any{
			"liveChatAuthorBadgeRenderer",
			"customThumbnail",
			"thumbnails",
			-1,
			"url",
		}, true)
		if ok {
			imgSrc = customUrl.(string)
		}

		chatBadges = append(chatBadges, ChatUserBadge{
			Name:   name.(string),
			ImgSrc: imgSrc,
			Type:   badgeType.(string),
		})
	}

	return chatBadges
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
		"image",
		"accessibility",
		"accessibilityData",
		"label",
	}, true)
	if ok {
		message.Text = emojiText.(string)
	}
	emojiImg, ok := GetDeepMapValue(messageEntry, []any{
		"emoji",
		"image",
		"thumbnails",
		-1,
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
		emojiName = emojiText
	}
	message.EmoteName = emojiName.(string)

	if message.Text == "" {
		message.Text = message.EmoteName
	}

	return message, nil
}

func getMessagesAPIResponse(continuationToken string) (map[string]any, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
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
	resp, err := client.Post(
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

func getUserIdFromURL(streamUrl string) (string, error) {
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
	index := strings.Index(string(body), "\"externalChannelId\":\"")
	if index < 0 {
		return "", &CustomError{message: "User ID not found in response"}
	}
	index += 21 // Length of the string before the user ID value
	endIndex := strings.Index(string(body)[index:], "\"")
	if endIndex < 0 {
		return "", &CustomError{message: "End of user ID not found"}
	}

	return string(body)[index : index+endIndex], nil
}

func getContinuationFromAllChat(continuationToken string) (string, error) {
	parsed, err := getMessagesAPIResponse(continuationToken)
	if err != nil {
		return "", err
	}

	return getContinuationFromAPIResponse(parsed)
}

func getLiveStreamFromChannelId(channelID string) (string, error) {
	resp, err := getYoutubeInitialData(channelID)
	if err != nil {
		return "", err
	}

	videoId, ok := GetDeepMapValue(resp, []any{
		"contents",
		"twoColumnBrowseResultsRenderer",
		"tabs",
		0,
		"tabRenderer",
		"content",
		"sectionListRenderer",
		"contents",
		0,
		"itemSectionRenderer",
		"contents",
		0,
		"channelFeaturedContentRenderer",
		"items",
		0,
		"videoRenderer",
		"videoId",
	}, false)
	if !ok {
		return "", &CustomError{message: "Live stream video ID not found in channel data"}
	}

	return "https://www.youtube.com/watch?v=" + videoId.(string), nil
}

func getYoutubeInitialData(channelID string) (map[string]any, error) {
	resp, err := http.Get("https://www.youtube.com/@" + channelID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &CustomError{message: "Failed to fetch channel page, status code: " + strconv.Itoa(resp.StatusCode)}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Find the initial data script tag
	startIndex := strings.Index(string(body), "ytInitialData = ")
	if startIndex < 0 {
		return nil, &CustomError{message: "Initial data not found in response"}
	}
	endIndex := strings.Index(string(body)[startIndex:], "};") + 1
	if endIndex < 0 {
		return nil, &CustomError{message: "End of initial data not found"}
	}
	rawJson := string(body)[startIndex+16 : startIndex+endIndex]

	var data map[string]any
	err = json.Unmarshal([]byte(rawJson), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
