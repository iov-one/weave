package weave

import (
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

func unmarshalHex(dst *[]byte, src []byte) (err error) {
	var s string
	err = json.Unmarshal(src, &s)
	if err != nil {
		return errors.Wrap(err, "parse string")
	}
	// and interpret that string as hex
	*dst, err = hex.DecodeString(s)
	return err
}

func marshalHex(bytes []byte) ([]byte, error) {
	s := strings.ToUpper(hex.EncodeToString(bytes))
	return json.Marshal(s)
}
