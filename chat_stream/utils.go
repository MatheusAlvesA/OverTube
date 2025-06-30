package chat_stream

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
)

func logIfNotSilent(message string, silent bool) {
	if !silent {
		log.Println(message)
	}
}

func GetDeepMapValue(m map[string]any, keys []any, silent bool) (any, bool) {
	var value any = m
	for _, key := range keys {
		if reflect.TypeOf(key) == reflect.TypeOf("") {
			if vTest, ok := value.(map[string]any); !ok {
				logIfNotSilent("[GetDeepMapValue] Element is not a map: "+fmt.Sprintf("%T", vTest)+" for key: "+key.(string), silent)
				return nil, false
			}
			if _, ok := value.(map[string]any)[key.(string)]; !ok {
				logIfNotSilent("[GetDeepMapValue] Key not found in map: "+key.(string), silent)
				return nil, false
			}
			value = value.(map[string]any)[key.(string)]
		} else if reflect.TypeOf(key) == reflect.TypeOf(1) {
			if vTest, ok := value.([]any); !ok {
				logIfNotSilent("[GetDeepMapValue] Element is not a array: "+fmt.Sprintf("%T", vTest)+" for key: "+key.(string), silent)
				return nil, false
			}
			if key.(int) < 0 {
				key = len(value.([]any)) + key.(int) // Handle negative indices
			}
			if key.(int) >= len(value.([]any)) {
				logIfNotSilent("[GetDeepMapValue] Key not found in array: "+strconv.Itoa(key.(int)), silent)
				return nil, false
			}
			value = value.([]any)[key.(int)]
		} else {
			logIfNotSilent("[GetDeepMapValue] Key is not a valid type:"+fmt.Sprintf("%T", key), silent)
			return nil, false
		}
	}
	return value, true
}
