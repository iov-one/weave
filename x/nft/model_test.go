package nft_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
)

func TestNonFungibleTokenValidate(t *testing.T) {
	alice := weavetest.NewCondition()
	specs := []struct {
		token    nft.NonFungibleToken
		expError bool
	}{
		{ // happy path
			token: nft.NonFungibleToken{
				ID:    []byte("anyID"),
				Owner: alice.Address(),
			},
			expError: false,
		},
		{ // not an address
			token: nft.NonFungibleToken{
				ID:    []byte("anyID"),
				Owner: []byte("not an address"),
			},
			expError: true,
		},
		{ // id to small
			token: nft.NonFungibleToken{
				ID:    []byte("12"),
				Owner: alice.Address(),
			},
			expError: true,
		},
		{ // id too big
			token: nft.NonFungibleToken{
				ID:    anyIDWithLength(257),
				Owner: alice.Address(),
			},
			expError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.token.Validate()
			if spec.expError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
