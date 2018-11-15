package client

import "encoding/json"

// ToString is a generic stringer which outputs
// a struct in its equivalent (indented) json representation
func ToString(d interface{}) string {
	s, err := json.MarshalIndent(d, "", "	")
	if err != nil {
		return err.Error()
	}
	return string(s)
}
