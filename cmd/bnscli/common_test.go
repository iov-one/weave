package main

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func fromBase64(t testing.TB, raw string) []byte {
	t.Helper()

	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("cannot decode base64 encoded data: %s", err)
	}
	return b
}

func fromHex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
