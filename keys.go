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

// DecodePrivateKey reads a hex string created by EncodePrivateKey
// and returns the original PrivateKey
func DecodePrivateKey(hexKey string) (*crypto.PrivateKey, error) {
	data, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	var key crypto.PrivateKey
	err = key.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// EncodePrivateKey stores the private key as a hex string
// that can be saved and later loaded
func EncodePrivateKey(key *crypto.PrivateKey) (string, error) {
	data, err := key.Marshal()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// LoadPrivateKey will load a private key from a file,
// Which was previously writen by SavePrivateKey
func LoadPrivateKey(filename string) (*crypto.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return DecodePrivateKey(string(raw))
}

// SavePrivateKey will encode the privatekey in hex and write to
// the named file. It will refuse to overwrite a file
func SavePrivateKey(key *crypto.PrivateKey, filename string) error {
	// don't overwrite some keys here...
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return fmt.Errorf("Refusing to overwrite: %s", filename)
	}
	return ForceSavePrivateKey(key, filename)
}

// ForceSavePrivateKey is like SavePrivateKey,
// except it will overwrite any file already present.
func ForceSavePrivateKey(key *crypto.PrivateKey, filename string) error {
	hexKey, err := EncodePrivateKey(key)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(hexKey), KeyPerm)
}
