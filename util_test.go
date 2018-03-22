package weave

import (
	"fmt"
	"testing"

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

type obj struct {
	bz []byte
}

// Marshal can return bz or error
func (a obj) Marshal() ([]byte, error) {
	if len(a.bz) < 2 {
		return nil, fmt.Errorf("Too short")
	}
	return a.bz, nil
}

func TestAddress(t *testing.T) {
	bad := Address{1, 3, 5}
	assert.Error(t, bad.Validate())

	// creating address
	bz := []byte("bling")
	one := &obj{bz}
	addr := NewAddress(bz)
	assert.NoError(t, addr.Validate())
	addr2 := MustObjAddress(one)
	assert.NoError(t, addr2.Validate())
	assert.True(t, addr.Equals(addr2))
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

	// bad address
	two := &obj{}
	assert.Panics(t, func() { MustObjAddress(two) })
}
