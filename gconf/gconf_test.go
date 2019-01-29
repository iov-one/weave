package gconf

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
)

func TestString(t *testing.T) {
	store := confStore(`"foobar"`)
	if want, got := "foobar", String(store, "a"); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestInt(t *testing.T) {
	store := confStore(`123`)
	if want, got := 123, Int(store, "a"); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDuration(t *testing.T) {
	store := confStore(`123`)
	if want, got := 123*time.Nanosecond, Duration(store, "a"); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestStrings(t *testing.T) {
	store := confStore(`["a", "b", "c"]`)
	if want, got := []string{"a", "b", "c"}, Strings(store, "a"); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestAddress(t *testing.T) {
	store := confStore(`"6161616161616161616161616161616161616161"`)
	if want, got := weave.Address(`aaaaaaaaaaaaaaaaaaaa`), Address(store, "a"); !got.Equals(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestBytes(t *testing.T) {
	store := confStore(`"YWJjZA=="`)
	if want, got := []byte("abcd"), Bytes(store, "a"); !bytes.Equal(got, want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestLoadingUnknownValuePanics(t *testing.T) {
	var recovered bool
	func() {
		defer func() {
			err := recover()
			recovered = err != nil
			t.Logf("recover(): %+v", err)
		}()

		loadInto(confStore(nil), "this-value-does-not-exist", nil)
	}()

	if !recovered {
		t.Fatal("expected loadInto call to panic")
	}
}

type confStore []byte

func (cs confStore) Get([]byte) []byte {
	return cs
}
