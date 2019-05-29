package client

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/iov-one/weave/crypto"
	"github.com/pkg/errors"
)

// KeyPerm is the file permissions for saved private keys
const KeyPerm = 0600

type PrivateKey = crypto.PrivateKey

// GenPrivateKey creates a new random key.
// Alias to simplify usage.
func GenPrivateKey() *PrivateKey {
	return crypto.GenPrivKeyEd25519()
}

func DecodePrivateKeyFromSeed(hexSeed string) (*PrivateKey, error) {
	data, err := hex.DecodeString(hexSeed)
	if err != nil {
		return nil, err
	}
	if len(data) != 64 {
		return nil, errors.New("invalid key")
	}
	key := &PrivateKey{Priv: &crypto.PrivateKey_Ed25519{Ed25519: data}}
	return key, nil
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
// Which was previously written by SavePrivateKey
func LoadPrivateKey(filename string) (*PrivateKey, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return DecodePrivateKey(string(raw))
}

// SavePrivateKey will encode the private key in hex and write to
// the named file
//
// Refuses to overwrite a file unless force is true
func SavePrivateKey(key *PrivateKey, filename string, force bool) error {
	if err := canWrite(filename, force); err != nil {
		return err
	}
	// actually do the write
	hexKey, err := EncodePrivateKey(key)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(hexKey), KeyPerm)
}

// LoadPrivateKeys will load an array of private keys from a file,
// Which was previously written by SavePrivateKeys
func LoadPrivateKeys(filename string) ([]*PrivateKey, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var encoded []string
	err = json.Unmarshal(raw, &encoded)
	if err != nil {
		return nil, err
	}

	keys := make([]*PrivateKey, len(encoded))
	for i, hexKey := range encoded {
		keys[i], err = DecodePrivateKey(hexKey)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

// SavePrivateKeys will encode an array of private keys
// as a json array of hex strings and
// write to the named file
//
// Refuses to overwrite a file unless force is true
func SavePrivateKeys(keys []*PrivateKey, filename string, force bool) error {
	var err error
	if err = canWrite(filename, force); err != nil {
		return err
	}
	encoded := make([]string, len(keys))
	for i, k := range keys {
		encoded[i], err = EncodePrivateKey(k)
		if err != nil {
			return err
		}
	}
	data, err := json.Marshal(encoded)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, KeyPerm)
}

// KeysByAddress takes a list of keys and creates a map
// to look up private keys by their (hex-encoded) address
func KeysByAddress(keys []*PrivateKey) map[string]*PrivateKey {
	res := make(map[string]*PrivateKey, len(keys))
	for _, k := range keys {
		addr := k.PublicKey().Address()
		res[addr.String()] = k
	}
	return res
}

// canWrite is a little helper to check if we want to write a file
func canWrite(filename string, force bool) error {
	if force {
		return nil
	}
	_, err := os.Stat(filename)
	if err == nil {
		return errors.Errorf("Refusing to overwrite: %s", filename)
	}
	return nil
}
