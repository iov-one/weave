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

func TestCreateSwapMsg(t *testing.T) {
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()
	validCoin := coin.NewCoin(1, 1, "TEST")
	invalidCoin := coin.NewCoin(1, 1, "12345789")

	specs := map[string]struct {
		Mutator func(msg *aswap.CreateSwapMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid hash": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.PreimageHash = make([]byte, 31)
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid recipient": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Recipient = nil
			},
			Exp: errors.ErrEmpty,
		},
		"Invalid src": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Src = nil
			},
			Exp: errors.ErrEmpty,
		},
		"0 timeout": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Timeout = 0
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid timeout": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Timeout = math.MinInt64
			},
			Exp: errors.ErrInvalidState,
		},
		"Invalid memo": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Memo = string(make([]byte, 129))
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid amount": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Amount = nil
			},
			Exp: errors.ErrInvalidAmount,
		},
		"Invalid coin": {
			Mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Amount = []*coin.Coin{&invalidCoin}
			},
			Exp: errors.ErrCurrency,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.CreateSwapMsg{Metadata: &weave.Metadata{Schema: 1},
			Src:          alice.Address(),
			Recipient:    bob.Address(),
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

func TestReleaseSwapMsg(t *testing.T) {
	specs := map[string]struct {
		Mutator func(msg *aswap.ReleaseSwapMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.ReleaseSwapMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid preimage": {
			Mutator: func(msg *aswap.ReleaseSwapMsg) {
				msg.Preimage = make([]byte, 31)
			},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid SwapID": {
			Mutator: func(msg *aswap.ReleaseSwapMsg) {
				msg.SwapID = nil
			},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.ReleaseSwapMsg{
			SwapID:   make([]byte, 1),
			Preimage: make([]byte, 32),
			Metadata: &weave.Metadata{Schema: 1},
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

func TestReturnSwapMsg(t *testing.T) {
	specs := map[string]struct {
		Mutator func(msg *aswap.ReturnSwapMsg)
		Exp     *errors.Error
	}{

		"Happy path": {},
		"Invalid metadata": {
			Mutator: func(msg *aswap.ReturnSwapMsg) {
				msg.Metadata.Schema = 0
			},
			Exp: errors.ErrMetadata,
		},
		"Invalid SwapID": {
			Mutator: func(msg *aswap.ReturnSwapMsg) {
				msg.SwapID = nil
			},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		baseMsg := aswap.ReturnSwapMsg{
			SwapID:   make([]byte, 1),
			Metadata: &weave.Metadata{Schema: 1},
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
