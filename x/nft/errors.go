package nft

import (
	"encoding/hex"
)

const (
	UnsupportedTokenType = "unsupported token type"
)

// id's are stored as bytes, but most are ascii text
// if in ascii, just convert to string
// if not, hex-encode it and prefix with 0x
func PrintableID(id []byte) string {
	if len(id) == 0 {
		return "<nil>"
	}
	if isSafeAscii(id) {
		return string(id)
	}
	return "0x" + hex.EncodeToString(id)
}

// require all bytes between 0x20 and 0x7f
func isSafeAscii(id []byte) bool {
	for _, c := range id {
		if c < 0x20 || c > 0x7f {
			return false
		}
	}
	return true
}
