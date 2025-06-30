package chat_stream

type PlatformType string

const (
	PlatformTypeYoutube PlatformType = "youtube"
	PlatformTypeTwitch  PlatformType = "twitch"
)

type CustomError struct {
	message string
}

func (e *CustomError) Error() string {
	return e.message
}

const ChatStreamMessageBufferSize = 200

type ChatStreamMessage struct {
	Platform  PlatformType
	Name      string
	Message   string
	Timestamp int64
}

type ChatStreamCon interface {
	IsConnected() bool
	GetMessagesChan() <-chan ChatStreamMessage
	Close()
}

type YTChatStreamCon struct {
	ChannelID         string
	ContinuationToken string
	LastStreamUpdate  int64
	stream            chan ChatStreamMessage
}

func (c *YTChatStreamCon) IsConnected() bool {
	// Placeholder for actual connection check logic
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
