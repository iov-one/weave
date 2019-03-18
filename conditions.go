package weave

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/iov-one/weave/crypto/bech32"
	"github.com/iov-one/weave/errors"
)

var (
	// AddressLength is the length of all addresses
	// You can modify it in init() before any addresses are calculated,
	// but it must not change during the lifetime of the kvstore
	AddressLength = 20

	// it must have (?s) flags, otherwise it errors when last section contains 0x20 (newline)
	perm = regexp.MustCompile(`(?s)^([a-zA-Z0-9_\-]{3,8})/([a-zA-Z0-9_\-]{3,8})/(.+)$`)
)

// Condition is a specially formatted array, containing
// information on who can authorize an action.
// It is of the format:
//
//   sprintf("%s/%s/%s", extension, type, data)
type Condition []byte

func NewCondition(ext, typ string, data []byte) Condition {
	pre := fmt.Sprintf("%s/%s/", ext, typ)
	return append([]byte(pre), data...)
}

// Parse will extract the sections from the Condition bytes
// and verify it is properly formatted
func (c Condition) Parse() (string, string, []byte, error) {
	chunks := perm.FindSubmatch(c)
	if len(chunks) == 0 {
		return "", "", nil, errors.ErrInvalidInput.Newf("condition: %X", []byte(c))

	}
	// returns [all, match1, match2, match3]
	return string(chunks[1]), string(chunks[2]), chunks[3], nil
}

// Address will convert a Condition into an Address
func (c Condition) Address() Address {
	return NewAddress(c)
}

// Equals checks if two permissions are the same
func (a Condition) Equals(b Condition) bool {
	return bytes.Equal(a, b)
}

// String returns a human readable string.
// We keep the extension and type in ascii and
// hex-encode the binary data
func (c Condition) String() string {
	ext, typ, data, err := c.Parse()
	if err != nil {
		return fmt.Sprintf("Invalid Condition: %X", []byte(c))
	}
	return fmt.Sprintf("%s/%s/%X", ext, typ, data)
}

// Validate returns an error if the Condition is not the proper format
func (c Condition) Validate() error {
	if !perm.Match(c) {
		return errors.ErrInvalidInput.Newf("condition: %X", []byte(c))
	}
	return nil
}

func (c Condition) MarshalJSON() ([]byte, error) {
	var serialized string
	if c != nil {
		serialized = c.String()
	}
	return json.Marshal(serialized)
}

func (c *Condition) UnmarshalJSON(raw []byte) error {
	var enc string
	if err := json.Unmarshal(raw, &enc); err != nil {
		return errors.Wrap(err, "cannot decode json")
	}
	return c.deserialize(enc)
}

// deserialize from human readable string.
func (c *Condition) deserialize(source string) error {
	// No value zero the address.
	if len(source) == 0 {
		*c = nil
		return nil
	}

	args := strings.Split(source, "/")
	if len(args) != 3 {
		return errors.ErrInvalidInput.Newf("invalid condition format")
	}
	data, err := hex.DecodeString(args[2])
	if err != nil {
		return errors.ErrInvalidInput.Newf("malformed condition data: %s", err)
	}
	*c = NewCondition(args[0], args[1], data)
	return nil
}

// Address represents a collision-free, one-way digest
// of a Condition
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
	s := strings.ToUpper(hex.EncodeToString(a))
	return json.Marshal(s)
}

func (a *Address) UnmarshalJSON(raw []byte) error {
	var enc string
	if err := json.Unmarshal(raw, &enc); err != nil {
		return errors.Wrap(err, "cannot decode json")
	}

	// If the encoded string starts with a prefix, cut it off and use
	// specified decoding method instead of default one.
	chunks := strings.SplitN(enc, ":", 2)
	format := chunks[0]
	if len(chunks) == 1 {
		format = "hex"
	} else {
		enc = chunks[1]
	}

	// No value zero the address.
	if len(enc) == 0 {
		*a = nil
		return nil
	}

	switch format {
	case "hex":
		val, err := hex.DecodeString(enc)
		if err != nil {
			return errors.Wrap(err, "cannot decode hex")
		}
		addr := Address(val)
		if err := Address(addr).Validate(); err != nil {
			return err
		}
		*a = val
		return nil
	case "cond":
		var c Condition
		if err := c.deserialize(enc); err != nil {
			return err
		}
		if err := c.Validate(); err != nil {
			return err
		}
		*a = c.Address()
		return nil
	case "bech32":
		_, payload, err := bech32.Decode(enc)
		if err != nil {
			return errors.Wrapf(err, "deserialize bech32: %s", err)
		}
		addr := Address(payload)
		if err := addr.Validate(); err != nil {
			return err
		}
		*a = addr
		return nil
	default:
		return errors.ErrInvalidType.Newf("unknown format %q", chunks[0])
	}
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
		return errors.ErrInvalidInput.Newf("address: %v", a)
	}
	return nil
}

// NewAddress hashes and truncates into the proper size
func NewAddress(data []byte) Address {
	if data == nil {
		return nil
	}
	// h := blake2b.Sum256(data)
	h := sha256.Sum256(data)
	return h[:AddressLength]
}
