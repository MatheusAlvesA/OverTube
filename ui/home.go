package ui

import (
	"image/color"
	"log"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func CreateHomeWindow(uiEvents chan<- UIEvent) {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("OverTube"))
		window.Option(app.MinSize(400, 300))
		err := run(window, uiEvents)
		uiEvents <- UIEventExit{err: err}
		close(uiEvents)
	}()
	app.Main()
}

func initialState() *UIState {
	state := new(UIState)
	state.YoutubeChannelURLEditor = &widget.Editor{
		SingleLine: true,
	}
	state.YouTubeChannelClickable = new(widget.Clickable)
	return state
}

func run(window *app.Window, uiEvents chan<- UIEvent) error {
	theme := material.NewTheme()
	state := initialState()
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			emitEvents(gtx, state, uiEvents)

			// Main component layout
			layout.Flex{
				Axis:      layout.Vertical,
				Spacing:   layout.SpaceEnd,
				Alignment: layout.Start,
			}.Layout(
				gtx,
				renderTitle(theme),
				renderYoutubeChannelInput(theme, state),
			)

			e.Frame(gtx.Ops)
		}
	}
}

func emitEvents(gtx layC, state *UIState, uiEvents chan<- UIEvent) {
	if state.YouTubeChannelClickable.Clicked(gtx) {
		log.Println("AQUI")
		uiEvents <- UIEventSetYoutubeChannel{
			Channel: state.YoutubeChannelURLEditor.Text(),
		}
	}
}

func renderTitle(theme *material.Theme) layout.FlexChild {
	title := material.H3(theme, "OverTube")

	maroon := color.NRGBA{R: 127, G: 0, B: 0, A: 255}
	title.Color = maroon

	title.Alignment = text.Start

	return layout.Rigid(func(gtx layC) layD {
		return title.Layout(gtx)
	})
}

func renderYoutubeChannelInput(
	theme *material.Theme,
	state *UIState,
) layout.FlexChild {
	editor := state.YoutubeChannelURLEditor
	editorUI := material.Editor(theme, editor, "YouTube Channel URL")

	submit := state.YouTubeChannelClickable
	submitUI := material.Button(theme, submit, "Set Channel")

	margin := layout.Inset{
		Top:    unit.Dp(16),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Bottom: unit.Dp(16),
	}

	return layout.Rigid(
		func(gtx layC) layD {
			return margin.Layout(gtx, func(gtx layC) layD {
				return layout.Flex{
					Axis:    layout.Horizontal,
					Spacing: layout.SpaceBetween,
				}.Layout(
					gtx,
					layout.Rigid(
						func(gtx layC) layD {
							return editorUI.Layout(gtx)
						},
					),
					layout.Rigid(
						func(gtx layC) layD {
							return submitUI.Layout(gtx)
						},
					),
				)
			})
		},
	)
}
