package save_state

import (
	"encoding/json"
	"log"
	"os"
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
		YoutubeChannel: "",
		TwitchChannel:  "",
	}

	dataJson, err := os.ReadFile("state.json")
	if err != nil {
		log.Println("[save_state::Read] Fail to read file", err)
		return defaultState
	}
	var readedState *AppState
	err = json.Unmarshal(dataJson, &readedState)
	if err != nil {
		log.Println("[save_state::Read] Fail parse file content", err)
		return defaultState
	}

	return readedState
}
