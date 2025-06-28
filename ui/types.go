package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
)

type layC = layout.Context
type layD = layout.Dimensions

type UIEvent interface {
	GetError() error
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
}
