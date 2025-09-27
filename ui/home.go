package ui

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"overtube/chat_stream"
	"overtube/save_state"
	"overtube/web_server"
	"overtube/ws_server"
	"regexp"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/sys/windows"
)

//go:embed platform_icons/*
var platformIcons embed.FS

func CreateHomeWindow(uiEvents chan<- UIEvent, uiCommands <-chan UICommand, appState *save_state.AppState) {
	go func() {
		window := &app.Window{}
		window.Option(app.Title("OverTube"))
		window.Option(app.MinSize(400, 300))
		err := run(window, uiEvents, uiCommands, appState)
		uiEvents <- UIEventExit{err: err}
		close(uiEvents)
	}()
	app.Main()
}

func initialState() *UIState {
	state := &UIState{}
	state.MainList = &widget.List{}
	state.MainList.Axis = layout.Vertical

	state.YoutubeChannelURLEditor = &widget.Editor{}
	state.YoutubeChannelURLEditor.SingleLine = true
	state.YoutubeChannelURLEditor.MaxLen = 60
	state.YouTubeChannelClickable = &widget.Clickable{}

	state.TwitchChannelURLEditor = &widget.Editor{}
	state.TwitchChannelURLEditor.SingleLine = true
	state.TwitchChannelURLEditor.MaxLen = 60
	state.TwitchChannelClickable = &widget.Clickable{}

	state.CopyLinkToChatClickable = &widget.Clickable{}
	state.VersionClickable = &widget.Clickable{}
	state.ConfirmCSSClickable = &widget.Clickable{}
	state.RevertCSSClickable = &widget.Clickable{}

	state.ChatStyleId = 1
	state.ChatStyleClickables = make(map[uint]*widget.Clickable)
	state.ChatStyleCustomCSSs = make(map[uint]*widget.Editor)
	for _, style := range web_server.GetChatStyleOptions() {
		state.ChatStyleClickables[style.Id] = &widget.Clickable{}
		editor := &widget.Editor{}
		editor.SingleLine = false // Permitir múltiplas linhas para CSS
		state.ChatStyleCustomCSSs[style.Id] = editor
	}

	return state
}

func readAppState(state *UIState, appState save_state.AppState) {
	if len(appState.YoutubeChannel) > 0 {
		state.YoutubeChannelURLEditor.SetText(appState.YoutubeChannel)
		state.YoutubeChannelSet = appState.YoutubeChannel
		state.YoutubeChannelWasConnected = true
	}
	if len(appState.TwitchChannel) > 0 {
		state.TwitchChannelURLEditor.SetText(appState.TwitchChannel)
		state.TwitchChannelSet = appState.TwitchChannel
		state.TwitchChannelWasConnected = true
	}
	if appState.ChatStyleId > 0 {
		state.ChatStyleId = appState.ChatStyleId
	}
	for _, css := range web_server.GetChatStyleOptions() {
		editor := &widget.Editor{}
		editor.SingleLine = false // Permitir múltiplas linhas para CSS
		state.ChatStyleCustomCSSs[css.Id] = editor
		state.ChatStyleCustomCSSs[css.Id].SetText(web_server.GetCurrentCSSForId(css.Id, &appState))
	}
}

func run(window *app.Window, uiEvents chan<- UIEvent, uiCommands <-chan UICommand, appState *save_state.AppState) error {
	theme := material.NewTheme()
	state := initialState()
	readAppState(state, *appState)
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
			state.MainList.Layout(gtx, 8, func(gtx layC, index int) layD {
				switch index {
				case 0:
					return renderTitle(gtx, theme, state)
				case 1:
					return renderYoutubeChannelInput(gtx, theme, state)
				case 2:
					return renderTwichChannelInput(gtx, theme, state)
				case 3:
					return renderBtnCopyLinkToChat(gtx, theme, state)
				case 4:
					return renderCustomSectionLineSeparator(gtx, theme)
				case 5:
					return renderCustomizeSection(gtx, theme, state)
				case 6:
					return renderCSSInputSection(gtx, theme, state)
				case 7:
					return renderCSSInputConfirmBtns(gtx, theme, state)
				default:
					return layout.Dimensions{}
				}
			})

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
	if state.YouTubeChannelClickable.Clicked(gtx) && validateYoutubeChannelURLEditor(state) {
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

	if state.TwitchChannelClickable.Clicked(gtx) && validateTwichChannelURLEditor(state) {
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

	if state.CopyLinkToChatClickable.Clicked(gtx) {
		gtx.Execute(clipboard.WriteCmd{Data: io.NopCloser(strings.NewReader("http://localhost:1337"))})
		state.CopyLinkToChatCopied = true
		go func() {
			time.Sleep(time.Second * 2)
			state.CopyLinkToChatCopied = false
		}()
	}

	if state.ConfirmCSSClickable.Clicked(gtx) {
		uiEvents <- SetChatStyleCustomCSS{
			Id:  state.ChatStyleId,
			CSS: state.GetChatStyleCustomCSS(state.ChatStyleId).Text(),
		}
	}

	if state.RevertCSSClickable.Clicked(gtx) {
		uiEvents <- ResetChatStyleCustomCSS{
			Id: state.ChatStyleId,
		}
		state.ChatStyleCustomCSSs[state.ChatStyleId].SetText(web_server.GetChatStyleFromId(state.ChatStyleId).CSS)
	}

	if state.VersionClickable.Clicked(gtx) {
		windows.ShellExecute(0, nil, windows.StringToUTF16Ptr("https://github.com/MatheusAlvesA/OverTube"), nil, nil, windows.SW_SHOWNORMAL)
	}

	if state.YouTubeChannelClickable.Hovered() ||
		state.TwitchChannelClickable.Hovered() ||
		state.CopyLinkToChatClickable.Hovered() ||
		state.VersionClickable.Hovered() {
		pointer.CursorPointer.Add(gtx.Ops)
	}

	for id, clickable := range state.ChatStyleClickables {
		if clickable.Clicked(gtx) {
			state.ChatStyleId = id
			uiEvents <- UIEventSetChatStyle{
				Id: id,
			}
		}
		if clickable.Hovered() {
			pointer.CursorPointer.Add(gtx.Ops)
		}
	}
}

func renderTitle(gtx layC, theme *material.Theme, state *UIState) layD {
	return layout.Inset{
		Top:    unit.Dp(0),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Bottom: unit.Dp(0),
	}.Layout(gtx, func(gtx layC) layD {
		title := material.H3(theme, "OverTube")
		maroon := color.NRGBA{R: 127, G: 0, B: 0, A: 255}
		title.Color = maroon
		title.Alignment = text.Start

		labelVersion := material.Label(theme, unit.Sp(12), "Versão 0.8.0")
		if state.VersionClickable.Hovered() {
			labelVersion.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
		} else {
			labelVersion.Color = color.NRGBA{R: 127, G: 127, B: 127, A: 255}
		}
		labelVersion.Alignment = text.End

		return layout.Flex{
			Axis:      layout.Horizontal,
			Spacing:   layout.SpaceBetween,
			Alignment: layout.Middle,
		}.Layout(gtx, layout.Rigid(func(gtx layC) layD {
			return title.Layout(gtx)
		}), layout.Rigid(func(gtx layC) layD {
			return state.VersionClickable.Layout(gtx, func(gtx layC) layD {
				return labelVersion.Layout(gtx)
			})
		}))
	})
}

func renderYoutubeChannelInput(
	gtx layC,
	theme *material.Theme,
	state *UIState,
) layD {
	editor := state.YoutubeChannelURLEditor
	editorUI := material.Editor(theme, editor, "YouTube @Channel")
	editorUI.LineHeight = 1.5

	submit := state.YouTubeChannelClickable
	submitUI := material.Button(theme, submit, "Definir")

	if state.YoutubeChannelSet != "" {
		submitUI.Text = "Remover"
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

	return margin.Layout(gtx, func(gtx layC) layD {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Spacing:   layout.SpaceBetween,
			Alignment: layout.Middle,
		}.Layout(
			gtx,
			layout.Rigid(
				func(gtx layC) layD {
					return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layC) layD {
						// Image of youtube logo
						file, err := platformIcons.Open("platform_icons/yt.png")
						if err != nil {
							log.Println("Error loading youtube logo:", err)
							// Fallback: create a red square if image loading fails
							img := image.NewRGBA(image.Rect(0, 0, 25, 25))
							draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, G: 0, B: 0, A: 255}}, image.Point{}, draw.Src)
							return widget.Image{Src: paint.NewImageOp(img)}.Layout(gtx)
						}
						defer file.Close()

						img, err := png.Decode(file)
						if err != nil {
							log.Println("Error decoding youtube logo:", err)
							// Fallback: create a red square if image decoding fails
							fallbackImg := image.NewRGBA(image.Rect(0, 0, 25, 25))
							draw.Draw(fallbackImg, fallbackImg.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, G: 0, B: 0, A: 255}}, image.Point{}, draw.Src)
							return widget.Image{Src: paint.NewImageOp(fallbackImg)}.Layout(gtx)
						}

						return widget.Image{
							Src:   paint.NewImageOp(img),
							Scale: 0.6,
						}.Layout(gtx)
					})
				},
			),
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
												Y: gtx.Dp(unit.Dp(20)),
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
						Max: image.Pt(20, 20),
					}.Op(gtx.Ops)

					c := color.NRGBA{R: 92, G: 184, B: 92, A: 255}
					if state.YoutubeConnStatus == ws_server.ChannelConnectionStarting {
						c = color.NRGBA{R: 255, G: 204, B: 0, A: 255}
					}
					if state.YoutubeConnStatus == ws_server.ChannelConnectionStopped {
						c = color.NRGBA{R: 204, G: 51, B: 0, A: 255}
					}

					paint.FillShape(gtx.Ops, c, circle)

					return layout.Dimensions{Size: image.Pt(25, 25)}
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
}

func renderTwichChannelInput(
	gtx layC,
	theme *material.Theme,
	state *UIState,
) layD {
	editor := state.TwitchChannelURLEditor
	editorUI := material.Editor(theme, editor, "Twitch username")
	editorUI.LineHeight = 1.5

	submit := state.TwitchChannelClickable
	submitUI := material.Button(theme, submit, "Definir")

	if state.TwitchChannelSet != "" {
		submitUI.Text = "Remover"
	}

	if editor.Text() == "" {
		submitUI.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
		submitUI.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	}

	margin := layout.Inset{
		Top:    unit.Dp(16),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Bottom: unit.Dp(16),
	}

	return margin.Layout(gtx, func(gtx layC) layD {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Spacing:   layout.SpaceBetween,
			Alignment: layout.Middle,
		}.Layout(
			gtx,
			layout.Rigid(
				func(gtx layC) layD {
					return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layC) layD {
						// Image of twitch logo
						file, err := platformIcons.Open("platform_icons/tw.png")
						if err != nil {
							log.Println("Error loading twitch logo:", err)
							// Fallback: create a red square if image loading fails
							img := image.NewRGBA(image.Rect(0, 0, 25, 25))
							draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, G: 0, B: 0, A: 255}}, image.Point{}, draw.Src)
							return widget.Image{Src: paint.NewImageOp(img)}.Layout(gtx)
						}
						defer file.Close()

						img, err := png.Decode(file)
						if err != nil {
							log.Println("Error decoding twitch logo:", err)
							// Fallback: create a red square if image decoding fails
							fallbackImg := image.NewRGBA(image.Rect(0, 0, 25, 25))
							draw.Draw(fallbackImg, fallbackImg.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, G: 0, B: 0, A: 255}}, image.Point{}, draw.Src)
							return widget.Image{Src: paint.NewImageOp(fallbackImg)}.Layout(gtx)
						}

						return widget.Image{
							Src:   paint.NewImageOp(img),
							Scale: 0.45,
						}.Layout(gtx)
					})
				},
			),
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
												Y: gtx.Dp(unit.Dp(20)),
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
						Max: image.Pt(20, 20),
					}.Op(gtx.Ops)

					c := color.NRGBA{R: 92, G: 184, B: 92, A: 255}
					if state.TwitchConnStatus == ws_server.ChannelConnectionStarting {
						c = color.NRGBA{R: 255, G: 204, B: 0, A: 255}
					}
					if state.TwitchConnStatus == ws_server.ChannelConnectionStopped {
						c = color.NRGBA{R: 204, G: 51, B: 0, A: 255}
					}

					paint.FillShape(gtx.Ops, c, circle)

					return layout.Dimensions{Size: image.Pt(25, 25)}
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
}

func renderBtnCopyLinkToChat(
	gtx layC,
	theme *material.Theme,
	state *UIState,
) layD {
	btnUI := material.Button(theme, state.CopyLinkToChatClickable, "Copiar link para o chat")

	if state.CopyLinkToChatCopied {
		btnUI.Text = "Copiado!"
	}

	return layout.Inset{
		Top:    unit.Dp(16),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Bottom: unit.Dp(16),
	}.Layout(gtx, func(gtx layC) layD {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Spacing:   layout.SpaceBetween,
			Alignment: layout.Middle,
		}.Layout(
			gtx,
			layout.Rigid(
				func(gtx layC) layD {
					return layout.Inset{Left: unit.Dp(8)}.Layout(
						gtx,
						func(gtx layC) layD {
							// Define largura fixa para o botão
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
							return btnUI.Layout(gtx)
						},
					)
				},
			),
		)
	})
}

func renderCustomSectionLineSeparator(gtx layC, theme *material.Theme) layD {
	title := material.Label(theme, unit.Sp(16), "Customização")
	title.Color = color.NRGBA{R: 127, G: 127, B: 127, A: 255}

	return layout.Flex{
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceStart,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layC) layD {
			return layout.Inset{
				Top:    unit.Dp(0),
				Left:   unit.Dp(16),
				Right:  unit.Dp(0),
				Bottom: unit.Dp(0),
			}.Layout(gtx, func(gtx layC) layD {
				return title.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layC) layD {
			return layout.Inset{
				Top:    unit.Dp(2),
				Left:   unit.Dp(16),
				Right:  unit.Dp(16),
				Bottom: unit.Dp(16),
			}.Layout(gtx, func(gtx layC) layD {
				paint.FillShape(gtx.Ops,
					color.NRGBA{R: 220, G: 220, B: 220, A: 255},
					clip.Rect{
						Max: image.Point{
							X: gtx.Constraints.Max.X,
							Y: gtx.Dp(unit.Dp(2)),
						},
					}.Op(),
				)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(2)))}
			})
		}),
	)
}

func validateTwichChannelURLEditor(state *UIState) bool {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	currentText := state.TwitchChannelURLEditor.Text()
	cleanedText := re.ReplaceAllString(currentText, "")
	state.TwitchChannelURLEditor.SetText(cleanedText)

	return cleanedText != ""
}

func validateYoutubeChannelURLEditor(state *UIState) bool {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	currentText := state.YoutubeChannelURLEditor.Text()
	cleanedText := re.ReplaceAllString(currentText, "")
	state.YoutubeChannelURLEditor.SetText(cleanedText)

	return cleanedText != ""
}
