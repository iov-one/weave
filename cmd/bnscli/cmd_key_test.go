package main

import (
	"testing"

	"golang.org/x/crypto/ed25519"
)

func TestKeygent(t *testing.T) {
	const mnemonic = `shy else mystery outer define there front bracket dawn honey excuse virus lazy book kiss cannon oven law coconut hedgehog veteran narrow great cage`

	// Result of this test can be verified using iov-core implementation
	// available at https://iov-one.github.io/token-finder/
	cases := map[string]string{
		"m/44'/234'/0'": "tiov1c3n70dph9m2jepszfmmh84pu75zuga3zrsd7jw",
		"m/44'/234'/1'": "tiov10lzv8v2lds7jvmkdt6t6khmhydr920r2yux8p9",
		"m/44'/234'/2'": "tiov18gwds8rx8cajav3m4lr5j98vlly9n8ms930z2l",
		"m/44'/234'/3'": "tiov1casuhjhjcqlxhlcfpqak5uccpqyajzp0nj3639",
		"m/44'/234'/4'": "tiov16rjld9tw88yrcc954cvvtnern576daunnn8jmn",
	}

	for path, bech := range cases {
		t.Run(path, func(t *testing.T) {
			priv, err := keygen(mnemonic, path)
			if err != nil {
				t.Fatalf("cannot generate key: %s", err)
			}
			b, err := toBech32("tiov", priv.Public().(ed25519.PublicKey))
			if err != nil {
				t.Fatalf("cannot serialize to bech32: %s", err)
			}
			if got := string(b); got != bech {
				t.Logf("want: %s", bech)
				t.Logf(" got: %s", got)
				t.Fatal("unexpected bech address")
			}
		})
	}
}
