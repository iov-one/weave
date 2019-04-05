package gconf

import (
	"bytes"
	"testing"
	"time"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestString(t *testing.T) {
	db := store.MemStore()
	assert.Nil(t, SetValue(db, "x", "foobar"))
	got := String(db, "x")
	if got != "foobar" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestInt(t *testing.T) {
	db := store.MemStore()
	assert.Nil(t, SetValue(db, "x", 851))
	got := Int(db, "x")
	if got != 851 {
		t.Fatalf("unexpected value: %d", got)
	}
}

func TestDuration(t *testing.T) {
	db := store.MemStore()
	assert.Nil(t, SetValue(db, "x", time.Hour+time.Minute))
	got := Duration(db, "x")
	if got != time.Hour+time.Minute {
		t.Fatalf("unexpected value: %s", got)
	}
}

func TestAddress(t *testing.T) {
	db := store.MemStore()
	val := hexDecode(t, "6161616161616161616161616161616161616161")
	assert.Nil(t, SetValue(db, "x", val))
	got := Address(db, "x")
	if !got.Equals(val) {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestBytes(t *testing.T) {
	db := store.MemStore()
	assert.Nil(t, SetValue(db, "x", []byte("abc123")))
	got := Bytes(db, "x")
	if !bytes.Equal(got, []byte("abc123")) {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestCoin(t *testing.T) {
	db := store.MemStore()
	val := coin.NewCoin(3, 4, "IOV")
	assert.Nil(t, SetValue(db, "x", val))
	got := Coin(db, "x")
	if !got.Equals(val) {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestLoadingUnknownValuePanics(t *testing.T) {
	db := store.MemStore()
	assert.Panics(t, func() {
		String(db, "this-value-does-not-exist")
	})
}

func TestLoadingWrongTypePanics(t *testing.T) {
	db := store.MemStore()
	assert.Nil(t, SetValue(db, "a-number", 87125))
	assert.Panics(t, func() {
		String(db, "a-number")
	})
}
