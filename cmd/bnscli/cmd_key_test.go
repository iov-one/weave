package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ed25519"
)

func TestKeygen(t *testing.T) {
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

func TestMnemonic(t *testing.T) {
	cases := map[string]struct {
		mnemonic string
		wantErr  bool
	}{

		"valid mnemonic 12 words": {
			mnemonic: "super bulk plunge better rookie donor reward obscure rescue type trade pelican",
			wantErr:  false,
		},
		"valid mnemonic 15 words": {
			mnemonic: "say debris entry orange grief deer train until flock scrub volume artist skill obscure immense",
			wantErr:  false,
		},
		"valid mnemonic 18 words": {
			mnemonic: "fetch height snow poverty space follow seven festival wasp pet asset tattoo cement twist exile trend bench eternal",
			wantErr:  false,
		},
		"valid mnemonic 21 words": {
			mnemonic: "increase shine pumpkin curtain trash cabbage juice canal ugly naive name insane indoor assault snap taxi casual unhappy buddy defense artefact",
			wantErr:  false,
		},
		"valid mnemonic 24 words": {
			mnemonic: "usage mountain noodle inspire distance lyrics caution wait mansion never announce biology squirrel guess key gain belt same matrix chase mom beyond model toy",
			wantErr:  false,
		},
		"additional whitespace around mnemonnic is ignored": {
			mnemonic: `
			forget
				rely tiny
			ostrich drop edit
			assault mechanic pony extend
			together twelve
				  observe bullet dream
		  short glide crack orchard exotic zero fly spice final
			`,
			wantErr: false,
		},
		"mnenomic that is valid in a language other than English (Italian)": {
			mnemonic: "acrobata acuto adagio addebito addome adeguato aderire adipe adottare adulare affabile affetto affisso affranto aforisma",
			wantErr:  true,
		},
		"mnenomic that is valid in a language other than English (Japanese)": {
			mnemonic: " あつかう あっしゅく あつまり あつめる あてな あてはまる あひる あぶら あぶる あふれる あまい あまど ",
			wantErr:  true,
		},
		"initially valid mnemonic that the last word was changed": {
			mnemonic: "super bulk plunge better rookie donor reward obscure rescue type trade trade",
			wantErr:  true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			_, err := keygen(tc.mnemonic, "m/44'/234'/0'")
			if hasErr := err != nil; hasErr != tc.wantErr {
				t.Fatalf("returned erorr value: %+v", err)
			}
		})
	}
}

func TestMnemonicWhitespaceIsIgnored(t *testing.T) {
	// The same mnemonic words formatted differently (different whitespace)
	// produce the same key.

	words := []string{
		"super", "bulk", "plunge", "better", "rookie", "donor",
		"reward", "obscure", "rescue", "type", "trade", "pelican",
	}

	// Standard mnemonic is created by separating words with a single space.
	stdMnemonic := strings.Join(words, " ")
	stdSeed := bip39.NewSeed(stdMnemonic, "")

	// All other mnemonics are non standard but somehow supported by this
	// bip39 implementation.
	mnemonics := []string{
		strings.Join(words, "\t"),
		strings.Join(words, " \n"),
		strings.Join(words, "  \t \n  "),
	}

	for i, m := range mnemonics {
		s := bip39.NewSeed(m, "")
		if !bytes.Equal(stdSeed, s) {
			t.Logf("reference: %x", stdSeed)
			t.Logf("      got: %x", s)
			t.Fatalf("seed %d is different than standard seed", i)
		}
	}
}
