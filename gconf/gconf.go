package gconf

import (
	"fmt"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
)

// Int returns an integer value stored under given name.
// This function panics if configuration cannot be acquired.
func Int(db weave.KVStore, propName string) int {
	var value int
	loadInto(db, propName, &value)
	return int(value)
}

// Duration returns a duration value stored under given name.
// This function panics if configuration cannot be acquired.
func Duration(db weave.KVStore, propName string) time.Duration {
	var value time.Duration
	loadInto(db, propName, &value)
	return value
}

// String returns a string value stored under given name.
// This function panics if configuration cannot be acquired.
func String(db weave.KVStore, propName string) string {
	var value string
	loadInto(db, propName, &value)
	return value
}

// Address returns an address value stored under given name.
// This function panics if configuration cannot be acquired.
func Address(db weave.KVStore, propName string) weave.Address {
	var value weave.Address
	loadInto(db, propName, &value)
	return value
}

// Bytes returns a bytes value stored under given name.
// This function panics if configuration cannot be acquired.
func Bytes(db weave.KVStore, propName string) []byte {
	value := make([]byte, 0, 128)
	loadInto(db, propName, &value)
	return value
}

// Coin returns a coin value strored under given name.
// This function panics if configuration cannot be acquired.
func Coin(db weave.KVStore, propName string) coin.Coin {
	var value coin.Coin
	loadInto(db, propName, &value)
	return value
}

func loadInto(db weave.KVStore, propName string, dest interface{}) {
	if err := defaultConfBucket.Load(db, propName, dest); err != nil {
		msg := fmt.Sprintf("cannot load %q configuration: %s", propName, err)
		panic(msg)
	}
}

var defaultConfBucket = NewConfBucket()
