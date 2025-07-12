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

func (e UIEventSetYoutubeChannel) GetError() error { return nil }

type UIState struct {
	YoutubeChannelURLEditor *widget.Editor
	YouTubeChannelClickable *widget.Clickable
	YoutubeChannelSet       string
	YoutubeConnStatus       ws_server.ChannelConnectionStatus
}
