package web_server

import (
	"log"
	"overtube/save_state"
)

func GetChatStyleOptions() []ChatStyleOption {
	return []ChatStyleOption{
		{
			Id:    1,
			Label: "Default",
			CSS:   getCss("default"),
		},
		{
			Id:    2,
			Label: "Preto e branco",
			CSS:   getCss("black-and-white"),
		},
		{
			Id:    3,
			Label: "Simples",
			CSS:   getCss("simple"),
		},
		{
			Id:    4,
			Label: "Simples chique",
			CSS:   getCss("simple-chic"),
		},
	}
}

func getCss(name string) string {
	css, err := content.ReadFile("www/styles/" + name + ".css")
	if err != nil {
		log.Println(err)
		return ""
	}
	return string(css)
}

func GetChatStyleFromId(id uint) *ChatStyleOption {
	list := GetChatStyleOptions()
	for _, style := range list {
		if style.Id == id {
			return &style
		}
	}
	return nil
}

func GetCurrentCSSForId(id uint, appState *save_state.AppState) string {
	for _, css := range appState.ChatStyleCustomCSSs {
		if css.Id == id {
			return css.CSS
		}
	}
	return GetChatStyleFromId(id).CSS
}
