package weave

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestHex(t *testing.T) {
	cases := map[string]struct {
		orig    []byte
		ser     string
		invalid string
	}{
		"decimal": {[]byte{01, 02}, `"0102"`, `"012"`},
		"hex":     {[]byte{0xFF, 0x14, 0x56}, `"FF1456"`, `FF1456`},
		"empty":   {[]byte{}, `""`, `"`},
	}

	unmarshalHex := func(bz []byte, out *[]byte) (err error) {
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

	marshalHex := func(bytes []byte) ([]byte, error) {
		s := strings.ToUpper(hex.EncodeToString(bytes))
		return json.Marshal(s)
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			// marshal as expected
			bz, err := marshalHex(tc.orig)
			assert.Nil(t, err)
			ser := []byte(tc.ser)
			assert.Equal(t, ser, bz)

			// properly parse
			in := []byte{}
			err = unmarshalHex(ser, &in)
			assert.Nil(t, err)
			assert.Equal(t, tc.orig, in)

			// failure returns error and doesn't affect input
			err = unmarshalHex([]byte(tc.invalid), &in)
			assert.Equal(t, true, err != nil)
			assert.Equal(t, tc.orig, in)
		})
	}
}

func TestAddress(t *testing.T) {
	bad := Address{1, 3, 5}
	assert.Equal(t, true, bad.Validate() != nil)

	// creating address
	bz := []byte("bling")
	addr := NewAddress(bz)
	assert.Nil(t, addr.Validate())
	assert.Equal(t, false, addr.Equals(bz))
	assert.Equal(t, false, addr.Equals(bad))

	// marshalling
	foo := addr.String()
	assert.Equal(t, 2*AddressLength, len(foo))
	ser, err := addr.MarshalJSON()
	assert.Nil(t, err)
	addr3 := Address{}
	err = addr3.UnmarshalJSON(ser)
	assert.Nil(t, err)
	assert.Equal(t, true, addr.Equals(addr3))
}

func TestCondition(t *testing.T) {
	other := NewCondition("some", "such", []byte("data"))
	failure, err := hex.DecodeString("736967732F656432353531392F16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18")
	assert.Nil(t, err)
	data, err := hex.DecodeString("16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18")
	assert.Nil(t, err)

	cases := map[string]struct {
		perm    Condition
		isError bool
		ext     string
		typ     string
		data    []byte
		serial  string
	}{
		"bad format without data separator": {
			[]byte("fo6/ds2qa"), true, "", "", nil, "",
		},
		"invalid ext format": {
			NewCondition("a.b", "dfr", []byte{34}), true, "", "", nil, "",
		},
		"good format": {
			[]byte("Foo/B4r/BZZ"),
			false,
			"Foo",
			"B4r",
			[]byte("BZZ"),
			"Foo/B4r/425A5A",
		},
		"non-ascii data": {
			NewCondition("help", "W1N", []byte{0xCA, 0xFE}),
			false,
			"help",
			"W1N",
			[]byte{0xCA, 0xFE},
			"help/W1N/CAFE",
		},
		// some weird failure from random test case
		// turns out to do with 0xa (newline) character in data
		"including newline character": {
			failure,
			false,
			"sigs",
			"ed25519",
			data,
			"sigs/ed25519/16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18",
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			ext, typ, data, err := tc.perm.Parse()
			if tc.isError {
				assert.Equal(t, true, err != nil)
				assert.Equal(t, true, tc.perm.Validate() != nil)
				return
			}
			// make sure parse matches
			assert.Nil(t, err)
			assert.Nil(t, tc.perm.Validate())
			assert.Equal(t, tc.ext, ext)
			assert.Equal(t, tc.typ, typ)
			assert.Equal(t, tc.data, data)

			// equal should pass with proper bytes
			cp := NewCondition(ext, typ, data)
			assert.Equal(t, true, tc.perm.Equals(cp))

			// doesn't match arbitrary other permission
			assert.Equal(t, false, tc.perm.Equals(other))
			addr := tc.perm.Address()
			assert.Nil(t, addr.Validate())
			assert.Equal(t, false, other.Address().Equals(addr))

			// make sure we get expected string
			assert.Equal(t, tc.serial, tc.perm.String())
		})
	}
}

func TestEmpty(t *testing.T) {
	var addr Address
	var perm Condition
	badPerm := Condition{0xFA, 0xDE}

	assert.Equal(t, "(nil)", addr.String())
	assert.Nil(t, perm.Address())
	assert.Equal(t, "Invalid Condition: FADE", badPerm.String())
	assert.Equal(t, "Invalid Condition: ", perm.String())
}
