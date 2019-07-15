package multisig

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateContract(t *testing.T) {
	cases := map[string]struct {
		Contract *Contract
		WantErr  *errors.Error
	}{
		"valid contract": {
			Contract: &Contract{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
				ActivationThreshold: 1,
				AdminThreshold:      2,
				Address:             weavetest.NewCondition().Address(),
			},
			WantErr: nil,
		},
		"missing metadata": {
			Contract: &Contract{
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
				ActivationThreshold: 1,
				AdminThreshold:      2,
			},
			WantErr: errors.ErrMetadata,
		},
		"contract address is required": {
			Contract: &Contract{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
				ActivationThreshold: 1,
				AdminThreshold:      2,
				Address:             nil,
			},
			WantErr: errors.ErrEmpty,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Contract.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}

}
