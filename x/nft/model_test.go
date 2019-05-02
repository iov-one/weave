package nft_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/nft"
)

func TestNonFungibleTokenValidate(t *testing.T) {
	alice := weavetest.NewCondition()
	cases := map[string]struct {
		Token   nft.NonFungibleToken
		WantErr *errors.Error
	}{
		"valid model": {
			Token: nft.NonFungibleToken{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       []byte("anyID"),
				Owner:    alice.Address(),
			},
			WantErr: nil,
		},
		"missing metadata": {
			Token: nft.NonFungibleToken{
				ID:    []byte("anyID"),
				Owner: alice.Address(),
			},
			WantErr: errors.ErrMetadata,
		},
		"not an address": {
			Token: nft.NonFungibleToken{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       []byte("anyID"),
				Owner:    []byte("not an address"),
			},
			WantErr: errors.ErrInvalidInput,
		},
		"id to small": {
			Token: nft.NonFungibleToken{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       []byte("12"),
				Owner:    alice.Address(),
			},
			WantErr: errors.ErrInvalidInput,
		},
		"id too big": {
			Token: nft.NonFungibleToken{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       anyIDWithLength(257),
				Owner:    alice.Address(),
			},
			WantErr: errors.ErrInvalidInput,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Token.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}

func anyIDWithLength(n int) []byte {
	r := make([]byte, n)
	for i := 0; i < n; i++ {
		r[i] = byte('a')
	}
	return r
}
