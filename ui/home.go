package ui

import (
	"image"
	"image/color"
	"log"
	"overtube/chat_stream"
	"overtube/ws_server"
	"time"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func CreateHomeWindow(uiEvents chan<- UIEvent, uiCommands <-chan UICommand) {
	go func() {
		window := &app.Window{}
		window.Option(app.Title("OverTube"))
		window.Option(app.MinSize(400, 300))
		err := run(window, uiEvents, uiCommands)
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

	state.TwitchChannelURLEditor = &widget.Editor{}
	state.TwitchChannelURLEditor.SingleLine = true
	state.TwitchChannelURLEditor.MaxLen = 60
	state.TwitchChannelClickable = &widget.Clickable{}

	return state
}

func run(window *app.Window, uiEvents chan<- UIEvent, uiCommands <-chan UICommand) error {
	theme := material.NewTheme()
	state := initialState()
	go listenToCommands(window, state, uiCommands)
	go retryEngine(state, uiEvents)
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			state.UIClosed = true
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
				renderTwichChannelInput(theme, state),
			)

			e.Frame(gtx.Ops)
		}
	}
}

func listenToCommands(w *app.Window, state *UIState, uiCommands <-chan UICommand) {
	for {
		cmd, more := <-uiCommands
		if !more {
			log.Println("uiCommands event channel closed")
			break
		}
		handleCommand(w, state, cmd)
	}
}

func retryEngine(state *UIState, uiEvents chan<- UIEvent) {
	if state.UIClosed {
		log.Println("UI closed, stopping retry engine")
		return
	}

	if state.YoutubeChannelWasConnected &&
		state.YoutubeChannelSet != "" &&
		state.YoutubeConnStatus == ws_server.ChannelConnectionStopped {
		log.Println("Retrying connection to YouTube channel:", state.YoutubeChannelSet)
		uiEvents <- UIEventSetYoutubeChannel{
			Channel: state.YoutubeChannelSet,
		}
	}

	if state.TwitchChannelWasConnected &&
		state.TwitchChannelSet != "" &&
		state.TwitchConnStatus == ws_server.ChannelConnectionStopped {
		log.Println("Retrying connection to Twitch channel:", state.TwitchChannelSet)
		uiEvents <- UIEventSetTwitchChannel{
			Channel: state.TwitchChannelSet,
		}
	}

	time.Sleep(time.Second * 2)
	defer retryEngine(state, uiEvents)
}

func handleCommand(w *app.Window, state *UIState, cmd UICommand) {
	switch t := cmd.(type) {
	case ChannelConnectionStatusChange:
		if t.Platform == chat_stream.PlatformTypeYoutube {
			state.YoutubeConnStatus = t.Status
			if t.Status == ws_server.ChannelConnectionRunning {
				state.YoutubeChannelWasConnected = true
			}
			w.Invalidate()
		}
		if t.Platform == chat_stream.PlatformTypeTwitch {
			state.TwitchConnStatus = t.Status
			if t.Status == ws_server.ChannelConnectionRunning {
				state.TwitchChannelWasConnected = true
			}
			w.Invalidate()
		}
	}
}

func emitEvents(gtx layC, state *UIState, uiEvents chan<- UIEvent) {
	if state.YouTubeChannelClickable.Clicked(gtx) {
		if state.YoutubeChannelSet == "" {
			state.YoutubeChannelSet = state.YoutubeChannelURLEditor.Text()
			uiEvents <- UIEventSetYoutubeChannel{
				Channel: state.YoutubeChannelURLEditor.Text(),
			}
		} else {
			state.YoutubeChannelWasConnected = false
			state.YoutubeChannelSet = ""
			uiEvents <- UIEventRemoveYoutubeChannel{}
		}
	}

	if state.TwitchChannelClickable.Clicked(gtx) {
		if state.TwitchChannelSet == "" {
			state.TwitchChannelSet = state.TwitchChannelURLEditor.Text()
			uiEvents <- UIEventSetTwitchChannel{
				Channel: state.TwitchChannelURLEditor.Text(),
			}
		} else {
			state.TwitchChannelWasConnected = false
			state.TwitchChannelSet = ""
			uiEvents <- UIEventRemoveTwitchChannel{}
		}
	}

	if state.YouTubeChannelClickable.Hovered() || state.TwitchChannelClickable.Hovered() {
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
	editorUI := material.Editor(theme, editor, "YouTube @Channel")
	editorUI.LineHeight = 1.5

	submit := state.YouTubeChannelClickable
	submitUI := material.Button(theme, submit, "Set Channel")

	if state.YoutubeChannelSet != "" {
		submitUI.Text = "Remove"
	}

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
										if state.YoutubeChannelSet != "" {
											paint.FillShape(gtx.Ops,
												color.NRGBA{R: 220, G: 220, B: 220, A: 255},
												clip.Rect{
													Max: image.Point{
														X: gtx.Constraints.Max.X,
														Y: 30,
													},
												}.Op(),
											)
											return material.Body1(theme, state.YoutubeChannelSet).Layout(gtx)
										}
										return editorUI.Layout(gtx)
									},
								)
							})
						},
					),
					layout.Rigid(
						func(gtx layC) layD {
							circle := clip.Ellipse{
								Min: image.Pt(10, 10),
								Max: image.Pt(30, 30),
							}.Op(gtx.Ops)

							c := color.NRGBA{R: 92, G: 184, B: 92, A: 255}
							if state.YoutubeConnStatus == ws_server.ChannelConnectionStarting {
								c = color.NRGBA{R: 255, G: 204, B: 0, A: 255}
							}
							if state.YoutubeConnStatus == ws_server.ChannelConnectionStopped {
								c = color.NRGBA{R: 204, G: 51, B: 0, A: 255}
							}

							paint.FillShape(gtx.Ops, c, circle)

							return layout.Dimensions{Size: image.Pt(40, 40)}
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

func renderTwichChannelInput(
	theme *material.Theme,
	state *UIState,
) layout.FlexChild {
	editor := state.TwitchChannelURLEditor
	editorUI := material.Editor(theme, editor, "Twitch username of channel")
	editorUI.LineHeight = 1.5

	submit := state.TwitchChannelClickable
	submitUI := material.Button(theme, submit, "Set Channel")

	if state.TwitchChannelSet != "" {
		submitUI.Text = "Remove"
	}

	if state.TwitchChannelURLEditor.Text() == "" {
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
										if state.TwitchChannelSet != "" {
											paint.FillShape(gtx.Ops,
												color.NRGBA{R: 220, G: 220, B: 220, A: 255},
												clip.Rect{
													Max: image.Point{
														X: gtx.Constraints.Max.X,
														Y: 30,
													},
												}.Op(),
											)
											return material.Body1(theme, state.TwitchChannelSet).Layout(gtx)
										}
										return editorUI.Layout(gtx)
									},
								)
							})
						},
					),
					layout.Rigid(
						func(gtx layC) layD {
							circle := clip.Ellipse{
								Min: image.Pt(10, 10),
								Max: image.Pt(30, 30),
							}.Op(gtx.Ops)

							c := color.NRGBA{R: 92, G: 184, B: 92, A: 255}
							if state.TwitchConnStatus == ws_server.ChannelConnectionStarting {
								c = color.NRGBA{R: 255, G: 204, B: 0, A: 255}
							}
							if state.TwitchConnStatus == ws_server.ChannelConnectionStopped {
								c = color.NRGBA{R: 204, G: 51, B: 0, A: 255}
							}

							paint.FillShape(gtx.Ops, c, circle)

							return layout.Dimensions{Size: image.Pt(40, 40)}
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
