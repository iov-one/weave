package aswap_test

import (
	"math"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/aswap"
)

func TestCreateMsg(t *testing.T) {
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()
	validCoin := coin.NewCoin(1, 1, "TEST")
	invalidCoin := coin.NewCoin(1, 1, "12345789")

	specs := map[string]struct {
		Mutator func(msg *aswap.CreateMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid hash": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.PreimageHash = make([]byte, 31)
			},
			Exp: errors.ErrInput,
		},
		"Invalid destination": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Destination = nil
			},
			Exp: errors.ErrEmpty,
		},
		"Invalid src": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Source = nil
			},
			Exp: errors.ErrEmpty,
		},
		"0 timeout": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Timeout = 0
			},
			Exp: errors.ErrInput,
		},
		"Invalid timeout": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Timeout = math.MinInt64
			},
			Exp: errors.ErrState,
		},
		"Invalid memo": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Memo = string(make([]byte, 129))
			},
			Exp: errors.ErrInput,
		},
		"Invalid amount": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Amount = nil
			},
			Exp: errors.ErrAmount,
		},
		"Invalid coin": {
			Mutator: func(msg *aswap.CreateMsg) {
				msg.Amount = []*coin.Coin{&invalidCoin}
			},
			Exp: errors.ErrCurrency,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.CreateMsg{Metadata: &weave.Metadata{Schema: 1},
			Source:       alice.Address(),
			Destination:  bob.Address(),
			PreimageHash: make([]byte, 32),
			Amount:       []*coin.Coin{&validCoin},
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

func TestReleaseMsg(t *testing.T) {
	specs := map[string]struct {
		Mutator func(msg *aswap.ReleaseMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.ReleaseMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid preimage": {
			Mutator: func(msg *aswap.ReleaseMsg) {
				msg.Preimage = make([]byte, 31)
			},
			Exp: errors.ErrInput,
		},
		"Invalid SwapID": {
			Mutator: func(msg *aswap.ReleaseMsg) {
				msg.SwapID = make([]byte, 7)
			},
			Exp: errors.ErrInput,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.ReleaseMsg{
			Preimage: make([]byte, 32),
			Metadata: &weave.Metadata{Schema: 1},
			SwapID:   make([]byte, 8),
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

func TestReturnMsg(t *testing.T) {
	specs := map[string]struct {
		Mutator func(msg *aswap.ReturnMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.ReturnMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid SwapID": {
			Mutator: func(msg *aswap.ReturnMsg) {
				msg.SwapID = make([]byte, 7)
			},
			Exp: errors.ErrInput,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.ReturnMsg{
			Metadata: &weave.Metadata{Schema: 1},
			SwapID:   make([]byte, 8),
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
