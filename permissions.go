package weave

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/confio/weave/errors"
)

var (
	// AddressLength is the length of all addresses
	// You can modify it in init() before any addresses are calculated,
	// but it must not change during the lifetime of the kvstore
	AddressLength = 20

	perm = regexp.MustCompile(`^([a-zA-Z0-9_\-]{3,8})/([a-zA-Z0-9_\-]{3,8})/(.+)$`)
)

// Permission is a specially formatted array, containing
// information on who can authorize an action.
// It is of the format:
//
//   sprintf("%s/%s/%s", extension, type, data)
type Permission []byte

func NewPermission(ext, typ string, data []byte) Permission {
	pre := fmt.Sprintf("%s/%s/", ext, typ)
	return append([]byte(pre), data...)
}

// Parse will extract the sections from the Permission bytes
// and verify it is properly formatted
func (p Permission) Parse() (string, string, []byte, error) {
	chunks := perm.FindSubmatch(p)
	if len(chunks) == 0 {
		return "", "", nil, errors.ErrUnrecognizedPermission(p)
	}
	// returns [all, match1, match2, match3]
	return string(chunks[1]), string(chunks[2]), chunks[3], nil
}

// Hash will convert a Permission into an Address
func (p Permission) Hash() Address {
	return NewAddress(p)
}

// Equals checks if two permissions are the same
func (a Permission) Equals(b Permission) bool {
	return bytes.Equal(a, b)
}

// String returns a human readable string.
// We keep the extension and type in ascii and
// hex-encode the binary data
func (p Permission) String() string {
	ext, typ, data, err := p.Parse()
	if err != nil {
		return fmt.Sprintf("Invalid Permission: %x", []byte(p))
	}
	return fmt.Sprintf("%s/%s/%X", ext, typ, data)
}

// Validate returns an error if the Permission is not the proper format
func (p Permission) Validate() error {
	if !perm.Match(p) {
		return errors.ErrUnrecognizedPermission(p)
	}
	return nil
}

// Address represents a collision-free, one-way digest
// of a permission
//
// It will be of size AddressLength
type Address []byte

// Equals checks if two addresses are the same
func (a Address) Equals(b Address) bool {
	return bytes.Equal(a, b)
}

// MarshalJSON provides a hex representation for JSON,
// to override the standard base64 []byte encoding
func (a Address) MarshalJSON() ([]byte, error) {
	return marshalHex(a)
}

// UnmarshalJSON parses JSON in hex representation,
// to override the standard base64 []byte encoding
func (a *Address) UnmarshalJSON(src []byte) error {
	dst := (*[]byte)(a)
	return unmarshalHex(src, dst)
}

// String returns a human readable string.
// Currently hex, may move to bech32
func (a Address) String() string {
	if len(a) == 0 {
		return "(nil)"
	}
	return strings.ToUpper(hex.EncodeToString(a))
}

// Validate returns an error if the address is not the valid size
func (a Address) Validate() error {
	if len(a) != AddressLength {
		return errors.ErrUnrecognizedAddress(a)
	}
	return nil
}

// NewAddress hashes and truncates into the proper size
func NewAddress(data []byte) Address {
	// h := blake2b.Sum256(data)
	h := sha256.Sum256(data)
	return h[:AddressLength]
}
