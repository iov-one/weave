package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/validators"
)

func TestCmdSetValidators(t *testing.T) {
	cases := map[string]struct {
		Initial []weave.ValidatorUpdate
		Args    []string
		WantSet []weave.ValidatorUpdate
		// Because we use fmt.Errorf we cannot check the instance. Only
		// the occurrence.
		WantErr bool
	}{
		"set from scratch": {
			Args: []string{
				"-pubkey", "j4JRVstX",
				"-power", "4",
			},
			WantSet: []weave.ValidatorUpdate{
				{
					PubKey: weave.PubKey{
						Type: "ed25519",
						Data: fromBase64(t, "j4JRVstX"),
					},
					Power: 4,
				},
			},
		},
		"append to an existing": {
			Initial: []weave.ValidatorUpdate{
				{
					PubKey: weave.PubKey{
						Type: "ed25519",
						Data: fromBase64(t, "aaJRVstX"),
					},
					Power: 2,
				},
			},
			Args: []string{
				"-pubkey", "j4JRVstX",
				"-power", "4",
			},
			WantSet: []weave.ValidatorUpdate{
				{
					PubKey: weave.PubKey{
						Type: "ed25519",
						Data: fromBase64(t, "aaJRVstX"),
					},
					Power: 2,
				},
				{
					PubKey: weave.PubKey{
						Type: "ed25519",
						Data: fromBase64(t, "j4JRVstX"),
					},
					Power: 4,
				},
			},
		},
		"missing pubkey": {
			Args:    []string{},
			WantErr: true,
		},
		"invalid pubkey": {
			Args: []string{
				"-pubkey", "NOT-A-BASE64-ENCODED-VALUE",
			},
			WantErr: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var input bytes.Buffer
			if tc.Initial != nil {
				tx := bnsd.Tx{
					Sum: &bnsd.Tx_ValidatorsApplyDiffMsg{
						ValidatorsApplyDiffMsg: &validators.ApplyDiffMsg{
							Metadata:         &weave.Metadata{Schema: 1},
							ValidatorUpdates: tc.Initial,
						},
					},
				}
				if _, err := writeTx(&input, &tx); err != nil {
					t.Fatalf("cannot marshal transaction: %s", err)
				}
			}

			var output bytes.Buffer
			err := cmdSetValidators(&input, &output, tc.Args)

			if tc.WantErr && err == nil {
				t.Fatal("expected an error, but the call was successful")
			} else if !tc.WantErr && err != nil {
				t.Fatalf("unexpected error: cmd failed: %s", err)
			}
		})
	}
}
