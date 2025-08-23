package save_state

type ChatStyleCustomCSS struct {
	Id  uint
	CSS string
}

type AppState struct {
	YoutubeChannel      string
	TwitchChannel       string
	ChatStyleId         uint
	ChatStyleCustomCSSs []ChatStyleCustomCSS
}

func (s *AppState) SetChatStyleCustomCSS(id uint, css string) {
	for i, opt := range s.ChatStyleCustomCSSs {
		if opt.Id == id {
			s.ChatStyleCustomCSSs[i].CSS = css
			return
		}
	}
	s.ChatStyleCustomCSSs = append(s.ChatStyleCustomCSSs, ChatStyleCustomCSS{
		Id:  id,
		CSS: css,
	})
}

func (s *AppState) ResetChatStyleCustomCSS(id uint) {
	filtered := []ChatStyleCustomCSS{}
	for _, opt := range s.ChatStyleCustomCSSs {
		if opt.Id != id {
			filtered = append(filtered, opt)
		}
	}
	s.ChatStyleCustomCSSs = filtered
}
