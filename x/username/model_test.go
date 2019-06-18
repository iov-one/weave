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
				BlockchainID: strings.Repeat("x", 4),
				Address:      bytes.Repeat([]byte("x"), 1),
			},
			WantErr: nil,
		},
		"correct maximal length": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 32),
				Address:      bytes.Repeat([]byte("x"), 128),
			},
			WantErr: nil,
		},
		"blockchain too short": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 3),
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
				BlockchainID: strings.Repeat("x", 6),
				Address:      bytes.Repeat([]byte("x"), 0),
			},
			WantErr: errors.ErrInput,
		},
		"address too long": {
			BA: BlockchainAddress{
				BlockchainID: strings.Repeat("x", 6),
				Address:      bytes.Repeat([]byte("x"), 129),
			},
			WantErr: errors.ErrInput,
		},
		"blockchain ID cannot contain emoji": {
			BA: BlockchainAddress{
				BlockchainID: "😄😊😉😍😘😚😜😝😳😁",
				Address:      bytes.Repeat([]byte("x"), 32),
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

func TestUsernameTokenValidate(t *testing.T) {
	cases := map[string]struct {
		UsernameToken UsernameToken
		WantErr       *errors.Error
	}{
		"correct": {
			UsernameToken: UsernameToken{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "blockchain", Address: []byte("123456789")},
				},
				Owner: weavetest.NewCondition().Address(),
			},
			WantErr: nil,
		},
		"target missing ": {
			UsernameToken: UsernameToken{
				Metadata: &weave.Metadata{Schema: 1},
				Targets:  nil,
				Owner:    weavetest.NewCondition().Address(),
			},
			WantErr: errors.ErrEmpty,
		},
		"owner missing ": {
			UsernameToken: UsernameToken{
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
			if err := tc.UsernameToken.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
