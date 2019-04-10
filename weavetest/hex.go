package weavetest

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/iov-one/weave"
)

// RandomAddr returns a valid random weave address genearted on the fly.
func RandomAddr(t testing.TB) weave.Address {
	raw := make([]byte, weave.AddressLength)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("cannot generate a random address: %s", err)
	}
	a := weave.Address(raw)
	if err := a.Validate(); err != nil {
		t.Fatalf("generated address is not a valid weave address: %s", err)
	}
	return a
}

// DecodeAddr takes a hex encoded address string and returns it's raw
// representation as a weave address. This function ensures that returned value
// is a valid address.
func DecodeAddr(t testing.TB, encoded string) weave.Address {
	t.Helper()
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("cannot decode hex string: %s", err)
	}
	a := weave.Address(raw)
	if err := a.Validate(); err != nil {
		t.Fatalf("decoded string is not a valid address: %s", err)
	}
	return a
}
