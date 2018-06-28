package utils

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/confio/weave/crypto"
)

// KeyPerm is the file permissions for saved private keys
const KeyPerm = 0600

type PrivateKey = crypto.PrivateKey

// GenPrivateKey creates a new random key.
// Alias to simplify usage.
func GenPrivateKey() *PrivateKey {
	return crypto.GenPrivKeyEd25519()
}

// DecodePrivateKey reads a hex string created by EncodePrivateKey
// and returns the original PrivateKey
func DecodePrivateKey(hexKey string) (*PrivateKey, error) {
	data, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	var key PrivateKey
	err = key.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// EncodePrivateKey stores the private key as a hex string
// that can be saved and later loaded
func EncodePrivateKey(key *PrivateKey) (string, error) {
	data, err := key.Marshal()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// LoadPrivateKey will load a private key from a file,
// Which was previously writen by SavePrivateKey
func LoadPrivateKey(filename string) (*PrivateKey, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return DecodePrivateKey(string(raw))
}

// SavePrivateKey will encode the privatekey in hex and write to
// the named file. It will refuse to overwrite a file
func SavePrivateKey(key *PrivateKey, filename string, force bool) error {
	if !force { // check before overwriting keys
		_, err := os.Stat(filename)
		if err == nil {
			return fmt.Errorf("Refusing to overwrite: %s", filename)
		}
	}

	// actually do the write
	hexKey, err := EncodePrivateKey(key)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(hexKey), KeyPerm)
}
