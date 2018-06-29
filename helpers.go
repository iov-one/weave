package utils

import "encoding/json"

func ToString(d interface{}) string {
	s, err := json.MarshalIndent(d, "", "	")
	if err != nil {
		return err.Error()
	}
	return string(s)
}
