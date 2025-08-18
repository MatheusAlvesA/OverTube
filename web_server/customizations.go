package web_server

import "log"

func GetChatStyleOptions() []ChatStyleOption {
	css, err := content.ReadFile("www/styles/default.css")
	if err != nil {
		log.Println(err)
		return []ChatStyleOption{}
	}
	return []ChatStyleOption{
		{
			id:    1,
			label: "Default",
			css:   string(css),
		},
	}
}

func GetChatStyleFromId(id uint) *ChatStyleOption {
	list := GetChatStyleOptions()
	for _, style := range list {
		if style.id == id {
			return &style
		}
	}
	return nil
}
