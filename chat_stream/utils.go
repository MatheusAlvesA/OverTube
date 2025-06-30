package chat_stream

import (
	"log"
	"reflect"
)

func GetDeepMapValue(m map[string]any, keys []any) (any, bool) {
	var value any = m
	for _, key := range keys {
		if reflect.TypeOf(key) == reflect.TypeOf("") {
			if vTest, ok := value.(map[string]any); !ok {
				log.Println("[GetDeepMapValue] Element is not a map: ", reflect.TypeOf(vTest), " for key: ", key)
				return nil, false
			}
			if _, ok := value.(map[string]any)[key.(string)]; !ok {
				log.Println("[GetDeepMapValue] Key not found in map:", key)
				return nil, false
			}
			value = value.(map[string]any)[key.(string)]
		} else if reflect.TypeOf(key) == reflect.TypeOf(1) {
			if vTest, ok := value.([]any); !ok {
				log.Println("[GetDeepMapValue] Element is not a array: ", reflect.TypeOf(vTest), " for key: ", key)
				return nil, false
			}
			if key.(int) < 0 {
				key = len(value.([]any)) + key.(int) // Handle negative indices
			}
			if key.(int) >= len(value.([]any)) {
				log.Println("[GetDeepMapValue] Key not found in array:", key)
				return nil, false
			}
			value = value.([]any)[key.(int)]
		} else {
			log.Println("[GetDeepMapValue] Key is not a valid type:", reflect.TypeOf(key))
			return nil, false
		}
	}
	return value, true
}
