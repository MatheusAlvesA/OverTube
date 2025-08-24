package save_state

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
)

func Save(data *AppState) bool {
	dataJson, err := json.Marshal(data)
	if err != nil {
		log.Println("[save_state::Save] Fail to transform state into Json", err)
		return false
	}

	err = os.WriteFile("state.json", dataJson, 0666)
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func Read() *AppState {
	defaultState := &AppState{
		YoutubeChannel:      "",
		TwitchChannel:       "",
		ChatStyleId:         1,
		ChatStyleCustomCSSs: []ChatStyleCustomCSS{},
	}

	dataJson, err := os.ReadFile("state.json")
	if err != nil {
		log.Println("[save_state::Read] Fail to read file", err)
		return defaultState
	}
	var readedData map[string]any
	err = json.Unmarshal(dataJson, &readedData)
	if err != nil {
		log.Println("[save_state::Read] Fail parse file content", err)
		return defaultState
	}

	readedState := &AppState{
		YoutubeChannel:      getDataOrDefault(readedData, "YoutubeChannel", "").(string),
		TwitchChannel:       getDataOrDefault(readedData, "TwitchChannel", "").(string),
		ChatStyleId:         uint(getDataOrDefault(readedData, "ChatStyleId", float64(1)).(float64)),
		ChatStyleCustomCSSs: getCSSCustoms(readedData),
	}

	return readedState
}

func getDataOrDefault(readedData map[string]any, key string, defaultValue any) any {
	if readedData[key] == nil {
		return defaultValue
	}
	return readedData[key]
}

func getCSSCustoms(readedData map[string]any) []ChatStyleCustomCSS {
	if readedData["ChatStyleCustomCSSs"] == nil {
		return []ChatStyleCustomCSS{}
	}
	t := reflect.TypeOf(readedData["ChatStyleCustomCSSs"])
	if t.Kind() != reflect.Slice {
		log.Println("[save_state::Read] ChatStyleCustomCSSs is not a slice", t.Kind())
		return []ChatStyleCustomCSS{}
	}
	list := make([]ChatStyleCustomCSS, 0)
	for _, item := range readedData["ChatStyleCustomCSSs"].([]any) {
		list = append(list, ChatStyleCustomCSS{
			Id:  uint(item.(map[string]any)["Id"].(float64)),
			CSS: item.(map[string]any)["CSS"].(string),
		})
	}
	return list
}
