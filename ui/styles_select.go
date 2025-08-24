package ui

import (
	"image/color"
	"overtube/web_server"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func renderCustomizeSection(gtx layC, theme *material.Theme, state *UIState) layD {
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
}

func renderCSSInputSection(gtx layC, theme *material.Theme, state *UIState) layD {
	return layout.Inset{
		Top:    unit.Dp(16),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Bottom: unit.Dp(16),
	}.Layout(gtx, func(gtx layC) layD {
		return widget.Border{
			Color:        color.NRGBA{R: 200, G: 200, B: 200, A: 255},
			Width:        unit.Dp(1),
			CornerRadius: unit.Dp(4),
		}.Layout(gtx, func(gtx layC) layD {
			// Calcular altura para 4 linhas de texto
			lineHeight := gtx.Sp(theme.TextSize)
			maxHeight := lineHeight * 10

			// Aplicar restrição de altura
			gtx.Constraints.Max.Y = maxHeight
			gtx.Constraints.Min.Y = maxHeight

			editor := material.Editor(theme, state.GetChatStyleCustomCSS(state.ChatStyleId), "CSS")

			return editor.Layout(gtx)
		})
	})
}

func renderCSSInputConfirmBtns(gtx layC, theme *material.Theme, state *UIState) layD {
	confirm := state.ConfirmCSSClickable
	if confirm.Hovered() {
		pointer.CursorPointer.Add(gtx.Ops)
	}
	confirmUI := material.Button(theme, confirm, "Confirmar CSS")
	confirmUI.Background = color.NRGBA{R: 33, G: 155, B: 167, A: 255}
	confirmUI.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	revert := state.RevertCSSClickable
	if revert.Hovered() {
		pointer.CursorPointer.Add(gtx.Ops)
	}
	revertUI := material.Button(theme, revert, "Reverter CSS")
	revertUI.Background = color.NRGBA{R: 255, G: 165, B: 100, A: 255}
	revertUI.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceStart,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layC) layD {
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layC) layD {
				gtx.Constraints.Min.X = 0
				return revertUI.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layC) layD {
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layC) layD {
				gtx.Constraints.Min.X = 0
				return confirmUI.Layout(gtx)
			})
		}),
	)
}
