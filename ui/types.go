package ui

import (
	"overtube/chat_stream"
	"overtube/ws_server"

	"gioui.org/layout"
	"gioui.org/widget"
)

type layC = layout.Context
type layD = layout.Dimensions

type UIEvent interface {
	GetError() error
}

type UICommand interface {
	GetData() any
}

type ChannelConnectionStatusChange struct {
	Platform chat_stream.PlatformType
	Status   ws_server.ChannelConnectionStatus
}

func (c ChannelConnectionStatusChange) GetData() any {
	return c
}

type UIEventExit struct {
	err error
}

func (e UIEventExit) GetError() error { return e.err }

type UIEventSetYoutubeChannel struct {
	Channel string
}

type UIEventSetTwitchChannel struct {
	Channel string
}

func (e UIEventSetYoutubeChannel) GetError() error { return nil }

func (e UIEventSetTwitchChannel) GetError() error { return nil }

type UIEventRemoveYoutubeChannel struct{}
type UIEventRemoveTwitchChannel struct{}

func (e UIEventRemoveYoutubeChannel) GetError() error { return nil }
func (e UIEventRemoveTwitchChannel) GetError() error  { return nil }

type UIEventSetChatStyle struct {
	Id uint
}

func (e UIEventSetChatStyle) GetError() error { return nil }

type UIState struct {
	YoutubeChannelURLEditor    *widget.Editor
	YouTubeChannelClickable    *widget.Clickable
	YoutubeChannelSet          string
	YoutubeConnStatus          ws_server.ChannelConnectionStatus
	YoutubeChannelWasConnected bool

	TwitchChannelURLEditor    *widget.Editor
	TwitchChannelClickable    *widget.Clickable
	TwitchChannelSet          string
	TwitchConnStatus          ws_server.ChannelConnectionStatus
	TwitchChannelWasConnected bool

	CopyLinkToChatClickable *widget.Clickable
	CopyLinkToChatCopied    bool

	ChatStyleId         uint
	ChatStyleClickables map[uint]*widget.Clickable

	UIClosed bool
}

func (s *UIState) GetChatStyleClickable(id uint) *widget.Clickable {
	return s.ChatStyleClickables[id]
}
