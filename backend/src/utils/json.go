package utils

import "encoding/json"

func MustMarshalToString(v any) string {
	result, err := json.Marshal(v)

	if err != nil {
		panic(err)
	}

	return string(result)
}
