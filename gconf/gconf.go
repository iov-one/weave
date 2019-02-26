package gconf

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
)

type Store interface {
	Get([]byte) []byte
}

// Int returns an integer value stored under given name.
// This function panics if configuration cannot be acquired.
func Int(confStore Store, propName string) int {
	var value int
	loadInto(confStore, propName, &value)
	return value
}

// Duration returns a duration value stored under given name.
// This function panics if configuration cannot be acquired.
func Duration(confStore Store, propName string) time.Duration {
	var value time.Duration
	loadInto(confStore, propName, &value)
	return value
}

// String returns a string value stored under given name.
// This function panics if configuration cannot be acquired.
func String(confStore Store, propName string) string {
	var value string
	loadInto(confStore, propName, &value)
	return value
}

// Strings returns an array of string value stored under given name.
// This function panics if configuration cannot be acquired.
func Strings(confStore Store, propName string) []string {
	var value []string
	loadInto(confStore, propName, &value)
	return value
}

// Address returns an address value stored under given name.
// This function panics if configuration cannot be acquired.
func Address(confStore Store, propName string) weave.Address {
	var value weave.Address
	loadInto(confStore, propName, &value)
	return value
}

// Bytes returns a bytes value stored under given name.
// This function panics if configuration cannot be acquired.
func Bytes(confStore Store, propName string) []byte {
	value := make([]byte, 0, 128)
	loadInto(confStore, propName, &value)
	return value
}

func Coin(confStore Store, propName string) coin.Coin {
	var value coin.Coin
	loadInto(confStore, propName, &value)
	return value
}

func loadInto(confStore Store, propName string, dest interface{}) {
	key := []byte("gconf:" + propName)
	raw := confStore.Get(key)
	if raw == nil {
		panic(fmt.Sprintf("cannot load %q configuration: not found", propName))
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		panic(fmt.Sprintf("cannot load %q configuration: %s", propName, err))
	}
}
