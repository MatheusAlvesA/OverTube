package chat_stream

import "github.com/gorilla/websocket"

type PlatformType string

const (
	PlatformTypeYoutube PlatformType = "youtube"
	PlatformTypeTwitch  PlatformType = "twitch"
)

type ChatStreamMessagePartType string

const (
	ChatStreamMessagePartTypeText  ChatStreamMessagePartType = "text"
	ChatStreamMessagePartTypeEmote ChatStreamMessagePartType = "emote"
)

type CustomError struct {
	message string
}

func (e *CustomError) Error() string {
	return e.message
}

const ChatStreamMessageBufferSize = 200

type ChatStreamMessagePart struct {
	PartType    ChatStreamMessagePartType
	Text        string
	EmoteImgUrl string
	EmoteName   string
}

type ChatStreamMessage struct {
	Platform     PlatformType
	Name         string
	MessageParts []ChatStreamMessagePart
	Timestamp    int64
	Badges       []ChatUserBadge
}

func (m *ChatStreamMessage) GetMessagePlainText() string {
	var messageText string
	for _, part := range m.MessageParts {
		messageText += part.Text
	}
	return messageText
}

type ChatUserBadge struct {
	Name   string
	ImgSrc string
	Type   string
}

type ChatStreamCon interface {
	IsConnected() bool
	GetMessagesChan() <-chan ChatStreamMessage
	Close()
	GetPlatform() PlatformType
}

type YTChatStreamCon struct {
	ChannelID         string
	ContinuationToken string
	LastStreamUpdate  int64
	stream            chan ChatStreamMessage
}

func (c *YTChatStreamCon) IsConnected() bool {
	return c.stream != nil
}
func (c *YTChatStreamCon) GetMessagesChan() <-chan ChatStreamMessage {
	return c.stream
}
func (c *YTChatStreamCon) Close() {
	if c.stream != nil {
		close(c.stream)
		c.stream = nil // Clear the stream to prevent further messages
	}
}
func (c *YTChatStreamCon) GetPlatform() PlatformType {
	return PlatformTypeYoutube
}

type TWChatStreamCon struct {
	ChannelID string
	ws        *websocket.Conn
	stream    chan ChatStreamMessage
	badgesDB  map[string]ChatUserBadge
}

func (c *TWChatStreamCon) IsConnected() bool {
	return c.stream != nil && c.ws != nil
}
func (c *TWChatStreamCon) GetMessagesChan() <-chan ChatStreamMessage {
	return c.stream
}
func (c *TWChatStreamCon) Close() {
	if c.stream != nil {
		close(c.stream)
		c.stream = nil
	}
	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
}
func (c *TWChatStreamCon) GetPlatform() PlatformType {
	return PlatformTypeTwitch
}
