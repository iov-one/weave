package utils

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneration(t *testing.T) {
	private := GenPrivateKey()
	private2 := GenPrivateKey()

	// make sure they are random and basic equality checks work
	assert.Equal(t, private, private)
	assert.NotEqual(t, private, private2)
	assert.Equal(t, private.PublicKey(), private.PublicKey())
	assert.NotEqual(t, private.PublicKey(), private2.PublicKey())
}

func TestEncodeDecode(t *testing.T) {
	private := GenPrivateKey()
	private2 := GenPrivateKey()

	enc, err := EncodePrivateKey(private)
	require.NoError(t, err)
	require.NotEmpty(t, enc)

	enc2, err := EncodePrivateKey(private2)
	require.NoError(t, err)
	require.NotEmpty(t, enc)

	assert.NotEqual(t, enc, enc2)

	dec, err := DecodePrivateKey(enc)
	require.NoError(t, err)
	assert.Equal(t, private, dec)

	dec2, err := DecodePrivateKey(enc2)
	require.NoError(t, err)
	assert.Equal(t, private2, dec2)

	// corrupt key should return error
	_, err = DecodePrivateKey(enc2[2:])
	assert.Error(t, err)
}

func TestSaveLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "tools-util-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, "foo.key")
	filename2 := filepath.Join(dir, "bar.key")

	private := GenPrivateKey()
	private2 := GenPrivateKey()

	// Save and load key
	err = SavePrivateKey(private, filename, false)
	require.NoError(t, err)
	loaded, err := LoadPrivateKey(filename)
	require.NoError(t, err)
	assert.Equal(t, private, loaded)

	// try to over-write, but fails
	err = SavePrivateKey(private2, filename, false)
	assert.Error(t, err)
	// can write to other location...
	err = SavePrivateKey(private2, filename2, false)
	require.NoError(t, err)

	// both keys stored separately
	loaded, err = LoadPrivateKey(filename)
	require.NoError(t, err)
	assert.Equal(t, private, loaded)
	loaded2, err := LoadPrivateKey(filename2)
	require.NoError(t, err)
	assert.Equal(t, private2, loaded2)

	// force over-write works
	err = SavePrivateKey(private2, filename, true)
	assert.NoError(t, err)
	loaded, err = LoadPrivateKey(filename)
	require.NoError(t, err)
	assert.Equal(t, private2, loaded)
}

func TestSaveLoadMultipleKeys(t *testing.T) {
	dir, err := ioutil.TempDir("", "tools-util-multikey")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, "foo.key")
	filename2 := filepath.Join(dir, "bar.key")

	private := GenPrivateKey()
	private2 := GenPrivateKey()
	private3 := GenPrivateKey()

	empty := []*PrivateKey{}
	one := []*PrivateKey{private}
	two := []*PrivateKey{private2, private3}

	// Save and load key
	err = SavePrivateKeys(empty, filename, false)
	require.NoError(t, err)
	loaded, err := LoadPrivateKeys(filename)
	require.NoError(t, err)
	assert.Equal(t, empty, loaded)

	// try to over-write, but fails
	err = SavePrivateKeys(one, filename, false)
	assert.Error(t, err)

	// can write to other location...
	err = SavePrivateKeys(one, filename2, false)
	require.NoError(t, err)
	loaded2, err := LoadPrivateKeys(filename2)
	require.NoError(t, err)
	assert.Equal(t, one, loaded2)

	// can handle multiple keys and overwrite
	err = SavePrivateKeys(two, filename2, true)
	require.NoError(t, err)
	loaded2, err = LoadPrivateKeys(filename2)
	require.NoError(t, err)
	assert.Equal(t, two, loaded2)
}

func TestKeysByAddress(t *testing.T) {
	private := GenPrivateKey()
	addr := private.PublicKey().Address().String()
	private2 := GenPrivateKey()
	addr2 := private2.PublicKey().Address().String()
	private3 := GenPrivateKey()
	addr3 := private3.PublicKey().Address().String()

	empty := []*PrivateKey{}
	one := []*PrivateKey{private}
	keys := []*PrivateKey{private, private2, private3}

	lookup := KeysByAddress(empty)
	assert.Equal(t, 0, len(lookup))

	lookup = KeysByAddress(one)
	assert.Equal(t, 1, len(lookup))
	assert.Equal(t, private, lookup[addr])
	assert.Nil(t, lookup[addr2])

	lookup = KeysByAddress(keys)
	assert.Equal(t, 3, len(lookup))
	assert.Equal(t, private, lookup[addr])
	assert.Equal(t, private2, lookup[addr2])
	assert.Equal(t, private3, lookup[addr3])
}

func TestDecodesCliKey(t *testing.T) {
	address, err := hex.DecodeString("eaff4c2151ed58c8a308528f5cccd105b3f16a33")
	require.NoError(t, err)

	encodedKey := "0a403b48c9fb3ce29e5780571661b0712d356f5c4195daa915c7c26fb53008085d5beb7f29afc78d6ab75bcb01e6949c3f3f1ba4f61448336ef3f830f5261e311081"

	key, err := DecodePrivateKey(encodedKey)
	require.NoError(t, err)
	assert.EqualValues(t, address, key.PublicKey().Address())
}
