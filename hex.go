package weave

import (
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

func unmarshalHex(bz []byte, out *[]byte) (err error) {
	var s string
	err = json.Unmarshal(bz, &s)
	if err != nil {
		return errors.Wrap(err, "parse string")
	}
	// and interpret that string as hex
	val, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	// only update object on success
	*out = val
	return nil
}

func marshalHex(bytes []byte) ([]byte, error) {
	s := strings.ToUpper(hex.EncodeToString(bytes))
	return json.Marshal(s)
}
