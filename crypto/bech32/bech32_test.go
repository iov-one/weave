package bech32

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestBench32EncodeDecode(t *testing.T) {
	// hex => bech32
	//
	// Can be generated using
	// $ hx=`openssl rand -hex 20`
	// $ bc=`bech32  -e -h tiov $hx`
	cases := map[string]string{
		"cf4e7c9d8a882a3d7b7c9d6ddf8d684daefa4f00": "tiov1ea88e8v23q4r67mun4kalrtgfkh05ncql389t7",
		"9fff6eb6d33a989d4e09a948c14366bdec1b2ca2": "tiov1nllkadkn82vf6nsf49yvzsmxhhkpkt9zaf60hc",
		"94192292ed7c1a71c7001354e93277a3deb11044": "tiov1jsvj9yhd0sd8r3cqzd2wjvnh500tzyzy6q99xd",
		"8e4447940810e1ab6d989c907489f8ca445e863a": "tiov13ezy09qgzrs6kmvcnjg8fz0cefz9ap36dvnynj",
		"74b1ab2bd2ce0f3938cfa4fb3a91c01d896fb177": "tiov1wjc6k27jec8njwx05nan4ywqrkyklvth80xftr",
		"ddeb2d37bcd1d3385b8f2f908cc9306f6863af58": "tiov1mh4j6dau68fnsku097ggejfsda5x8t6c58k4ty",
		"8795be838a466971a1187da36f2138fbcd83afa2": "tiov1s72maqu2ge5hrggc0k3k7gfcl0xc8tazvs2yqk",
		"638a652f9e13e325dd289987655e293aa7c5fea6": "tiov1vw9x2tu7z03jthfgnxrk2h3f82nutl4x0jrza9",
		"b5129d368d7cf10ff634271e1b404acc60ba640b": "tiov1k5ff6d5d0ncsla35yu0pksz2e3st5eqtrzhss0",
		"cb33f07c2766314598894a019f27ba3e9c55f275": "tiov1evelqlp8vcc5txyffgqe7fa686w9tun4vtjcjw",
		"e078271a9c009a9770465896c55c9c050598f2e9": "tiov1upuzwx5uqzdfwuzxtztv2hyuq5ze3uhfmer4gq",
		"8c9cb774769aa06ec72aa5d79a1b21ac39e7c1fd": "tiov13jwtwarkn2sxa3e25hte5xep4su70s0at36php",
		"cf50845e3411d3fec1bc79bab707aae7942a9fad": "tiov1eagggh35z8flasdu0xatwpa2u72z48ad9r8xpf",
		"cca7b0acb4595ed727d6e13049c46860710d1dfb": "tiov1ejnmpt95t90dwf7kuycyn3rgvpcs680m90qvh5",
		"41b6ba23313cf034f6fadc14379d5e49803154db": "tiov1gxmt5ge38ncrfah6ms2r0827fxqrz4xmcg8eu7",
		"7aa2d7d63726dcdaba7625e23f7d280c704de98a": "tiov1023d043hymwd4wnkyh3r7lfgp3cym6v20v7f9x",
		"4398e4a0f9bdf9972398c246d6d43fea3d885a8b": "tiov1gwvwfg8ehhuewguccfrdd4plag7csk5txgg6uy",
		"c66c158e1d782278987a012f874e379fc9a16998": "tiov1cekptrsa0q383xr6qyhcwn3hnly6z6vcn4n49h",
		"529c44809cee723f745818b9634859e73869873f": "tiov122wyfqyuaeer7azcrzukxjzeuuuxnpelx79mkt",
		"e6e5bc0f56dbb47633d1261e8b39300ee6f8b4e0": "tiov1umjmcr6kmw68vv73yc0gkwfspmn03d8qxr0vph",
	}

	for hx, bc := range cases {
		t.Run(hx, func(t *testing.T) {
			want, err := hex.DecodeString(hx)
			if err != nil {
				t.Fatalf("cannot decode hex: %s", err)
			}

			hrp, payload, err := Decode(bc)
			if err != nil {
				t.Fatalf("cannot decode bech32: %s", err)
			}

			if hrp != "tiov" {
				t.Fatalf("invalid hrp: %s", hrp)
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

			if string(raw) != bc {
				t.Fatalf("invalid encoding: %q", raw)
			}
		})
	}
}
