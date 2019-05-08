package aswap_test

import (
	"math"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/aswap"
)

func TestSwap(t *testing.T) {
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()

	specs := map[string]struct {
		Mutator func(msg *aswap.Swap)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.Swap) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid hash": {
			Mutator: func(msg *aswap.Swap) {
				msg.PreimageHash = make([]byte, 31)
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid recipient": {
			Mutator: func(msg *aswap.Swap) {
				msg.Recipient = nil
			},
			Exp: errors.ErrEmpty,
		},
		"Invalid src": {
			Mutator: func(msg *aswap.Swap) {
				msg.Src = nil
			},
			Exp: errors.ErrEmpty,
		},
		"0 timeout": {
			Mutator: func(msg *aswap.Swap) {
				msg.Timeout = 0
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid timeout": {
			Mutator: func(msg *aswap.Swap) {
				msg.Timeout = math.MinInt64
			},
			Exp: errors.ErrInvalidState,
		},
		"Invalid memo": {
			Mutator: func(msg *aswap.Swap) {
				msg.Memo = string(make([]byte, 129))
			},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.Swap{Metadata: &weave.Metadata{Schema: 1},
			Src:          alice.Address(),
			Recipient:    bob.Address(),
			PreimageHash: make([]byte, 32),
			Timeout:      weave.UnixTime(1),
			Memo:         "",
		}

		t.Run(msg, func(t *testing.T) {
			if spec.Mutator != nil {
				spec.Mutator(&baseMsg)
			}
			err := baseMsg.Validate()
			if !spec.Exp.Is(err) {
				t.Fatalf("check expected: %v  but got %+v", spec.Exp, err)
			}
		})
	}
}
