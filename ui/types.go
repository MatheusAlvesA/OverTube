package ui

import (
	"image"
	"overtube/chat_stream"
	"overtube/ws_server"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
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

type SetChatStyleCustomCSS struct {
	Id  uint
	CSS string
}

func (e SetChatStyleCustomCSS) GetError() error { return nil }

type ResetChatStyleCustomCSS struct {
	Id uint
}

func (e ResetChatStyleCustomCSS) GetError() error { return nil }

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

	VersionClickable *widget.Clickable

	ChatStyleId         uint
	ChatStyleClickables map[uint]*widget.Clickable
	ChatStyleCustomCSSs map[uint]*widget.Editor
	ConfirmCSSClickable *widget.Clickable
	RevertCSSClickable  *widget.Clickable

	MainList *widget.List

	UIClosed bool
}

func (s *UIState) GetChatStyleClickable(id uint) *widget.Clickable {
	return s.ChatStyleClickables[id]
}

func (s *UIState) GetChatStyleCustomCSS(id uint) *widget.Editor {
	return s.ChatStyleCustomCSSs[id]
}

// Flow implementa um container com quebra de linha
type Flow struct {
	Spacing unit.Dp
}

func (f Flow) Layout(gtx layout.Context, children ...layout.Widget) layout.Dimensions {
	spacing := gtx.Dp(f.Spacing)
	cs := gtx.Constraints
	x, y := spacing, 0
	rowHeight := 0

	var maxY int

	for _, child := range children {
		// mede o tamanho do filho
		macro := op.Record(gtx.Ops)
		dims := child(gtx)
		c := macro.Stop()

		if x+dims.Size.X+spacing > cs.Max.X {
			// quebra de linha
			x = spacing
			y += rowHeight + spacing
			rowHeight = 0
		}
		trans := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		c.Add(gtx.Ops)
		trans.Pop()

		// Atualiza a posição para o próximo elemento
		x += dims.Size.X + spacing
		if dims.Size.Y > rowHeight {
			rowHeight = dims.Size.Y
		}
		if y+rowHeight > maxY {
			maxY = y + rowHeight
		}
	}

	return layout.Dimensions{Size: image.Pt(cs.Max.X, maxY)}
}
