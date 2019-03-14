package bech32

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestBench32EncodeDecode(t *testing.T) {
	// bech32  -e -h tiov 746573742d7061796c6f6164
	const enc = `tiov1w3jhxapdwpshjmr0v9jqymqq4y`

	want, err := hex.DecodeString("746573742d7061796c6f6164")
	if err != nil {
		t.Fatal(err)
	}

	hrp, payload, err := Decode(enc)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(want, payload) {
		t.Logf("want %d", want)
		t.Logf("got  %d", payload)
		t.Fatal("invalid decode")
	}

	raw, err := Encode(hrp, payload)
	if err != nil {
		t.Fatalf("cannot encode: %s", err)
	}

	if string(raw) != enc {
		t.Fatalf("invalid encoding: %q", raw)
	}
}
