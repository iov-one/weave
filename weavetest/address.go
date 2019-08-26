package weavetest

import (
	"testing"

	"github.com/iov-one/weave"
)

// ParseAddress takes a weave address in a human readable format and returns
// its binary representation. This function is a test helper that is using
// weave.ParseAddress function functionality.
func ParseAddress(t testing.TB, encodedAddress string) weave.Address {
	t.Helper()

	addr, err := weave.ParseAddress(encodedAddress)
	if err != nil {
		t.Fatalf("cannot parse %q address: %s", encodedAddress, err)
	}
	return addr
}
