package utils

import (
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
