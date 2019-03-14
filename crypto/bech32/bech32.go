package bech32

import (
	"github.com/btcsuite/btcutil/bech32"
	"github.com/iov-one/weave/errors"
)

// Decode converts given bech32 encoded representation into raw payload and a
// human readable part.
func Decode(raw string) (string, []byte, error) {
	hrp, payload, err := bech32.Decode(raw)
	if err != nil {
		return "", nil, errors.Wrap(err, "bech32 decode")
	}
	payload, err = bech32.ConvertBits(payload, 5, 8, false)
	if err != nil {
		return "", nil, errors.Wrap(err, "convert bits")
	}
	return hrp, payload, nil
}

// Encode converts given bytes into bech32 encoded representation.
func Encode(hrp string, payload []byte) ([]byte, error) {
	payload, err := bech32.ConvertBits(payload, 8, 5, true)
	if err != nil {
		return nil, errors.Wrap(err, "convert bits")
	}
	raw, err := bech32.Encode(hrp, payload)
	if err != nil {
		return nil, errors.Wrap(err, "bech32 encode")
	}
	return []byte(raw), nil
}
