package storage

import (
	"encoding/json"
)

func Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
