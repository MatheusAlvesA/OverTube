package ui

import (
	"image/color"
	"overtube/web_server"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func renderCustomizeSection(theme *material.Theme, state *UIState) layout.FlexChild {
	return layout.Rigid(
		func(gtx layC) layD {
			list := web_server.GetChatStyleOptions()
			buttons := make([]layout.Widget, len(list))
			for i, style := range list {
				selected := state.ChatStyleId == style.Id
				clickable := state.GetChatStyleClickable(style.Id)
				if clickable.Hovered() {
					pointer.CursorPointer.Add(gtx.Ops)
				}
				clickableUI := material.Button(theme, clickable, style.Label)
				if selected {
					clickableUI.Background = color.NRGBA{R: 33, G: 155, B: 167, A: 255}
					clickableUI.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				} else {
					clickableUI.Background = color.NRGBA{R: 122, G: 218, B: 165, A: 255}
					clickableUI.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
				}
				buttons[i] = func(gtx layC) layD {
					gtx.Constraints.Min.X = 0
					return clickableUI.Layout(gtx)
				}
			}

			return Flow{Spacing: unit.Dp(8)}.Layout(gtx, buttons...)
		},
	)
}
