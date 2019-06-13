package main

import (
	"bytes"
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

func TestUnpackSequence(t *testing.T) {
	cases := map[string]struct {
		Raw     string
		WantErr bool
		Want    []byte
	}{
		"default encoding (decimal)": {
			Raw:     "123",
			WantErr: false,
			Want:    sequenceID(123),
		},
		"zero decimal value is not allowed": {
			Raw:     "0",
			WantErr: true,
		},
		"hex encoded value": {
			Raw:     "hex:" + hex.EncodeToString(sequenceID(1234567890)),
			WantErr: false,
			Want:    sequenceID(1234567890),
		},
		"too short, hex encoded value": {
			Raw:     "hex:3132330a",
			WantErr: true,
		},
		"too long, hex encoded value": {
			Raw:     "hex:" + hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
			WantErr: true,
		},
		"base64 encoded value": {
			Raw:     "base64:" + base64.StdEncoding.EncodeToString(sequenceID(1234567890)),
			WantErr: false,
			Want:    sequenceID(1234567890),
		},
		"too short, base64 encoded value": {
			Raw:     "base64:" + base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
			WantErr: true,
		},
		"unknown encoding (random string)": {
			Raw:     "x:_P1U_!RU)RQU_AU)FAf",
			WantErr: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			b, err := unpackSequence(tc.Raw)

			if tc.WantErr {
				if err == nil {
					t.Fatalf("want error, got %x", b)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if !bytes.Equal(b, tc.Want) {
					t.Fatalf("unexpected result: %x", b)
				}
			}
		})
	}
}
