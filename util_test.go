package weave

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHex(t *testing.T) {
	cases := []struct {
		orig    []byte
		ser     string
		invalid string
	}{
		{[]byte{01, 02}, `"0102"`, `"012"`},
		{[]byte{0xFF, 0x14, 0x56}, `"FF1456"`, `FF1456`},
		{[]byte{}, `""`, `"`},
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

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			// marshal as expected
			bz, err := marshalHex(tc.orig)
			require.NoError(t, err)
			ser := []byte(tc.ser)
			assert.Equal(t, ser, bz)

			// properly parse
			in := []byte{}
			err = unmarshalHex(ser, &in)
			require.NoError(t, err)
			assert.Equal(t, tc.orig, in)

			// failure returns error and doesn't affect input
			err = unmarshalHex([]byte(tc.invalid), &in)
			assert.Error(t, err)
			assert.Equal(t, tc.orig, in)
		})
	}
}

func TestAddress(t *testing.T) {
	bad := Address{1, 3, 5}
	assert.Error(t, bad.Validate())

	// creating address
	bz := []byte("bling")
	addr := NewAddress(bz)
	assert.NoError(t, addr.Validate())
	assert.False(t, addr.Equals(bz))
	assert.False(t, addr.Equals(bad))

	// marshalling
	foo := fmt.Sprintf("%s", addr)
	assert.Equal(t, 2*AddressLength, len(foo))
	ser, err := addr.MarshalJSON()
	require.NoError(t, err)
	addr3 := Address{}
	err = addr3.UnmarshalJSON(ser)
	require.NoError(t, err)
	assert.True(t, addr.Equals(addr3))
}

func TestCondition(t *testing.T) {
	other := NewCondition("some", "such", []byte("data"))
	failure, err := hex.DecodeString("736967732F656432353531392F16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18")
	require.NoError(t, err)
	data, err := hex.DecodeString("16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18")
	require.NoError(t, err)

	cases := []struct {
		perm    Condition
		isError bool
		ext     string
		typ     string
		data    []byte
		serial  string
	}{
		// bad format
		{
			[]byte("fo6/ds2qa"), true, "", "", nil, "",
		},
		// bad format
		{
			NewCondition("a.b", "dfr", []byte{34}), true, "", "", nil, "",
		},
		// good format
		{
			[]byte("Foo/B4r/BZZ"),
			false,
			"Foo",
			"B4r",
			[]byte("BZZ"),
			"Foo/B4r/425A5A",
		},
		// non-ascii data
		{
			NewCondition("help", "W1N", []byte{0xCA, 0xFE}),
			false,
			"help",
			"W1N",
			[]byte{0xCA, 0xFE},
			"help/W1N/CAFE",
		},
		// some weird failure from random test case
		// turns out to do with 0xa (newline) character in data
		{
			failure,
			false,
			"sigs",
			"ed25519",
			data,
			"sigs/ed25519/16E290A51B2B136C2C213884D03B8BAE483D6133F0A3D110FED3890E0A5A4E18",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ext, typ, data, err := tc.perm.Parse()
			if tc.isError {
				require.Error(t, err)
				require.Error(t, tc.perm.Validate())
				return
			}
			// make sure parse matches
			require.NoError(t, err)
			require.NoError(t, tc.perm.Validate())
			assert.Equal(t, tc.ext, ext)
			assert.Equal(t, tc.typ, typ)
			assert.Equal(t, tc.data, data)

			// equal should pass with proper bytes
			cp := NewCondition(ext, typ, data)
			assert.True(t, tc.perm.Equals(cp))

			// doesn't match arbitrary other permission
			assert.False(t, tc.perm.Equals(other))
			addr := tc.perm.Address()
			assert.NoError(t, addr.Validate())
			assert.NotEqual(t, addr, other.Address())

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
