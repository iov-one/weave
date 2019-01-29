package gconf

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iov-one/weave"
)

type Store interface {
	Get([]byte) []byte
}

func Int(confStore Store, propName string) int {
	var value int
	loadInto(confStore, propName, &value)
	return value
}

func Duration(confStore Store, propName string) time.Duration {
	var value time.Duration
	loadInto(confStore, propName, &value)
	return value
}

func String(confStore Store, propName string) string {
	var value string
	loadInto(confStore, propName, &value)
	return value
}

func Strings(confStore Store, propName string) []string {
	var value []string
	loadInto(confStore, propName, &value)
	return value
}

func Address(confStore Store, propName string) weave.Address {
	var value weave.Address
	loadInto(confStore, propName, &value)
	return value
}

func Bytes(confStore Store, propName string) []byte {
	value := make([]byte, 0, 128)
	loadInto(confStore, propName, &value)
	return value
}

func loadInto(confStore Store, propName string, dest interface{}) {
	key := []byte("gconf:" + propName)
	raw := confStore.Get(key)
	if raw == nil {
		fail(fmt.Sprintf("cannot load %q configuration: not found", propName))
		return
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		fail(fmt.Sprintf("cannot load %q configuration: %s", propName, err))
	}
}

var fail = func(msg string) {
	panic(msg)
}
