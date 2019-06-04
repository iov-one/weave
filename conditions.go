package weave

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
		return "", "", nil, errors.Wrapf(errors.ErrInput, "condition: %X", []byte(c))

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
	if len(c) == 0 {
		return errors.ErrEmpty
	}
	if !perm.Match(c) {
		return errors.Wrapf(errors.ErrInput, "condition: %X", []byte(c))
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
		return errors.Wrap(errors.ErrInput, "invalid condition format")
	}
	data, err := hex.DecodeString(args[2])
	if err != nil {
		return errors.Wrapf(errors.ErrInput, "malformed condition data: %s", err)
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
	val, err := ParseAddress(enc)
	if err != nil {
		return err
	}
	*a = val
	return nil
}

// ParseAddress accepts address in a string format and unmarshals it.
func ParseAddress(enc string) (Address, error) {
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
		return nil, nil
	}
	switch format {
	case "hex":
		val, err := hex.DecodeString(enc)
		if err != nil {
			return nil, errors.Wrap(err, "cannot decode hex")
		}
		addr := Address(val)
		if err := Address(addr).Validate(); err != nil {
			return nil, err
		}
		return val, nil
	case "cond":
		var c Condition
		if err := c.deserialize(enc); err != nil {
			return nil, err
		}
		if err := c.Validate(); err != nil {
			return nil, err
		}
		return c.Address(), nil
	case "seq":
		chunks := strings.Split(string(enc), "/")
		if len(chunks) != 3 {
			return nil, errors.Wrap(errors.ErrInput, "invalid condition format")
		}
		seqInt, err := strconv.Atoi(chunks[2])
		if err != nil {
			return nil, errors.Wrap(err, "sequence number is not a valid integer")
		}
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(seqInt))
		c := NewCondition(chunks[0], chunks[1], data)
		if err := c.Validate(); err != nil {
			return nil, err
		}
		return c.Address(), nil
	case "bech32":
		_, payload, err := bech32.Decode(enc)
		if err != nil {
			return nil, errors.Wrapf(err, "deserialize bech32: %s", err)
		}
		addr := Address(payload)
		if err := addr.Validate(); err != nil {
			return nil, err
		}
		return addr, nil
	default:
		return nil, errors.Wrapf(errors.ErrType, "unknown format %q", chunks[0])
	}
}

// Clone provides an independent copy of an address.
func (a Address) Clone() Address {
	if a == nil {
		return nil
	}
	cpy := make(Address, len(a))
	copy(cpy, a)
	return cpy
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
	if len(a) == 0 {
		return errors.ErrEmpty
	}
	if len(a) != AddressLength {
		return errors.Wrapf(errors.ErrInput, "invalid address length: %v", a)
	}
	return nil
}

// Set updates this address value to what is provided. This method implements
// flag.Value interface.
func (a *Address) Set(enc string) error {
	val, err := ParseAddress(enc)
	if err != nil {
		return nil
	}
	*a = val
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
