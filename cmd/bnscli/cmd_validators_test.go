package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/validators"
)

func TestCmdSetValidators(t *testing.T) {
	cases := map[string]struct {
		Initial []*validators.ValidatorUpdate
		Args    []string
		WantSet []*validators.ValidatorUpdate
	}{
		"set from scratch": {
			Args: []string{
				"-pubkey", "j4JRVstX",
				"-power", "4",
			},
			WantSet: []*validators.ValidatorUpdate{
				{
					Pubkey: validators.Pubkey{
						Type: "ed25519",
						Data: fromBase64(t, "j4JRVstX"),
					},
					Power: 4,
				},
			},
		},
		"append to an existing": {
			Initial: []*validators.ValidatorUpdate{
				{
					Pubkey: validators.Pubkey{
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
			WantSet: []*validators.ValidatorUpdate{
				{
					Pubkey: validators.Pubkey{
						Type: "ed25519",
						Data: fromBase64(t, "aaJRVstX"),
					},
					Power: 2,
				},
				{
					Pubkey: validators.Pubkey{
						Type: "ed25519",
						Data: fromBase64(t, "j4JRVstX"),
					},
					Power: 4,
				},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var input bytes.Buffer
			if tc.Initial != nil {
				tx := app.Tx{
					Sum: &app.Tx_SetValidatorsMsg{
						SetValidatorsMsg: &validators.SetValidatorsMsg{
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
			if err := cmdSetValidators(&input, &output, tc.Args); err != nil {
				t.Fatalf("cmd failed: %s", err)
			}
		})
	}
}
