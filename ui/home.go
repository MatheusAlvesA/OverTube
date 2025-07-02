package ui

import (
	"image/color"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func CreateHomeWindow(uiEvents chan<- UIEvent) {
	go func() {
		window := &app.Window{}
		window.Option(app.Title("OverTube"))
		window.Option(app.MinSize(400, 300))
		err := run(window, uiEvents)
		uiEvents <- UIEventExit{err: err}
		close(uiEvents)
	}()
	app.Main()
}

func initialState() *UIState {
	state := &UIState{}
	state.YoutubeChannelURLEditor = &widget.Editor{}
	state.YoutubeChannelURLEditor.SingleLine = true
	state.YoutubeChannelURLEditor.MaxLen = 60

	state.YouTubeChannelClickable = &widget.Clickable{}
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
		uiEvents <- UIEventSetYoutubeChannel{
			Channel: state.YoutubeChannelURLEditor.Text(),
		}
	}
	if state.YouTubeChannelClickable.Hovered() {
		pointer.CursorPointer.Add(gtx.Ops)
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
	editorUI.LineHeight = 1.5

	submit := state.YouTubeChannelClickable
	submitUI := material.Button(theme, submit, "Set Channel")

	if state.YoutubeChannelURLEditor.Text() == "" {
		submitUI.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
		submitUI.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	}

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
					Axis:      layout.Horizontal,
					Spacing:   layout.SpaceBetween,
					Alignment: layout.Middle,
				}.Layout(
					gtx,
					layout.Flexed(
						1,
						func(gtx layC) layD {
							return widget.Border{
								Color:        color.NRGBA{R: 200, G: 200, B: 200, A: 255},
								Width:        unit.Dp(1),
								CornerRadius: unit.Dp(4),
							}.Layout(gtx, func(gtx layC) layD {
								return layout.UniformInset(6).Layout(
									gtx,
									func(gtx layC) layD {
										return editorUI.Layout(gtx)
									},
								)
							})
						},
					),
					layout.Rigid(
						func(gtx layC) layD {
							return layout.Inset{Left: unit.Dp(8)}.Layout(
								gtx,
								func(gtx layC) layD {
									return submitUI.Layout(gtx)
								},
							)
						},
					),
				)
			})
		},
	)
}
