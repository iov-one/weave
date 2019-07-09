package username

import (
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestBlockchainAddressValidation(t *testing.T) {
	cases := map[string]struct {
		BA      BlockchainAddress
		WantErr *errors.Error
	}{
		"correct minimal length": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 4),
				Address:      strings.Repeat("x", 1),
			},
			WantErr: nil,
		},
		"correct maximal length": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 32),
				Address:      strings.Repeat("x", 128),
			},
			WantErr: nil,
		},
		"blockchain too short": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 3),
				Address:      strings.Repeat("x", 3),
			},
			WantErr: errors.ErrInput,
		},
		"blockchain too long": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 33),
				Address:      strings.Repeat("x", 3),
			},
			WantErr: errors.ErrInput,
		},
		"address too short": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 6),
				Address:      strings.Repeat("x", 0),
			},
			WantErr: errors.ErrInput,
		},
		"address too long": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 6),
				Address:      strings.Repeat("x", 129),
			},
			WantErr: errors.ErrInput,
		},
		"blockchain ID cannot contain emoji": {
			BA: BlockchainAddress{
				BlockchainID: "😄😊😉😍😘😚😜😝😳😁",
				Address:      strings.Repeat("x", 32),
			},
			WantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.BA.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestQueryByOwner(t *testing.T) {
	var retrievedTokens []Token

	db := store.MemStore()
	migration.MustInitPkg(db, "username")
	username := Username("alice*iov")

	token := Token{
		Metadata: &weave.Metadata{Schema: 1},
		Targets: []BlockchainAddress{
			{BlockchainID: "blockchain", Address: "123456789"},
		},
		Owner: weavetest.NewCondition().Address(),
	}

	b := NewTokenBucket()

	_, err := b.Put(db, username.Bytes(), &token)
	assert.Nil(t, err)

	_, err = b.ByIndex(db, "owner", token.Owner, &retrievedTokens)
	assert.Nil(t, err)

	if len(retrievedTokens) != 1 {
		t.Fatalf("Expected to retrieve one token, got %d", len(retrievedTokens))
	}

	assert.Equal(t, token, retrievedTokens[0])
}

func TestTokenValidate(t *testing.T) {
	cases := map[string]struct {
		Token   Token
		WantErr *errors.Error
	}{
		"correct": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "blockchain", Address: "123456789"},
				},
				Owner: weavetest.NewCondition().Address(),
			},
			WantErr: nil,
		},
		"empty targets ": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets:  nil,
				Owner:    weavetest.NewCondition().Address(),
			},
		},
		"owner missing ": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "blockchain", Address: "123456789"},
				},
				Owner: nil,
			},
			WantErr: errors.ErrEmpty,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Token.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
