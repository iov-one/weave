package username

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestBlockchainAddressValidation(t *testing.T) {
	cases := map[string]struct {
		BA      BlockchainAddress
		WantErr *errors.Error
	}{
		"correct minimal length": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 3),
				Address:      bytes.Repeat([]byte("x"), 3),
			},
			WantErr: nil,
		},
		"correct maximal length": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 32),
				Address:      bytes.Repeat([]byte("x"), 1024),
			},
			WantErr: nil,
		},
		"blockchain too short": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 2),
				Address:      bytes.Repeat([]byte("x"), 3),
			},
			WantErr: errors.ErrInput,
		},
		"blockchain too long": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 33),
				Address:      bytes.Repeat([]byte("x"), 3),
			},
			WantErr: errors.ErrInput,
		},
		"address too short": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 3),
				Address:      bytes.Repeat([]byte("x"), 2),
			},
			WantErr: errors.ErrInput,
		},
		"address too long": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 3),
				Address:      bytes.Repeat([]byte("x"), 1025),
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

func TestTokenValidate(t *testing.T) {
	cases := map[string]struct {
		Token   Token
		WantErr *errors.Error
	}{
		"correct": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "blockchain", Address: []byte("123456789")},
				},
				Owner: weavetest.NewCondition().Address(),
			},
			WantErr: nil,
		},
		"target missing ": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets:  nil,
				Owner:    weavetest.NewCondition().Address(),
			},
			WantErr: errors.ErrEmpty,
		},
		"owner missing ": {
			Token: Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "blockchain", Address: []byte("123456789")},
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
