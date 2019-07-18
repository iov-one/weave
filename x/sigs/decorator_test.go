package sigs

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestDecorator(t *testing.T) {
	chainID := "deco-rate"

	priv := weavetest.NewKey()

	tx := NewStdTx([]byte("art"))
	sig0, err := SignTx(priv, tx, chainID, 0)
	assert.Nil(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	assert.Nil(t, err)

	ctx := weave.WithChainID(context.Background(), chainID)

	incrSequence := func(db store.KVStore, d Decorator, h *SigCheckHandler) {
		tx.Signatures = []*StdSignature{sig0}
		_, err := d.Check(ctx, db, tx, h)
		assert.Nil(t, err)
	}

	cases := map[string]struct {
		setup            func(store.KVStore, Decorator, *SigCheckHandler)
		allowMissingSigs bool
		srcSign          []*StdSignature
		expCheckErr      *errors.Error
		expDeliverErr    *errors.Error
		expSigners       []weave.Condition
	}{
		"with no sigs": {
			expCheckErr:   errors.ErrUnauthorized,
			expDeliverErr: errors.ErrUnauthorized,
		},
		"single signature": {
			srcSign:    []*StdSignature{sig0},
			expSigners: []weave.Condition{priv.PublicKey().Condition()},
		},
		"with replay": {
			setup:         incrSequence,
			srcSign:       []*StdSignature{sig0},
			expCheckErr:   ErrInvalidSequence,
			expDeliverErr: ErrInvalidSequence,
		},
		"with next sequence": {
			setup:      incrSequence,
			srcSign:    []*StdSignature{sig1},
			expSigners: []weave.Condition{priv.PublicKey().Condition()},
		},
		"allowing none": {
			allowMissingSigs: true,
			expSigners:       []weave.Condition{},
		},
		"allowing none, with next sequence": {
			setup:            incrSequence,
			allowMissingSigs: true,
			srcSign:          []*StdSignature{sig1},
			expSigners:       []weave.Condition{priv.PublicKey().Condition()},
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "sigs")
			captureSigners := new(SigCheckHandler)

			d := NewDecorator()
			if tc.allowMissingSigs {
				d = d.AllowMissingSigs()
			}

			if tc.setup != nil {
				tc.setup(db, d, captureSigners)
			}
			cache := db.CacheWrap()
			tx.Signatures = tc.srcSign

			// when
			_, err := d.Check(ctx, cache, tx, captureSigners)
			if !tc.expCheckErr.Is(err) {
				t.Fatalf("unexpected errror: %+v", err)

			}

			cache.Discard()
			// and when
			if _, err := d.Deliver(ctx, cache, tx, captureSigners); !tc.expDeliverErr.Is(err) {
				t.Fatalf("unexpected deliver error: %+v", err)
			}

			if tc.expDeliverErr != nil {
				// If we expect an error than it make no sense to continue the flow.
				return
			}
			assert.Equal(t, tc.expSigners, captureSigners.Signers)
		})
	}

}

// SigCheckHandler stores the seen signers on each call
type SigCheckHandler struct {
	Signers []weave.Condition
}

var _ weave.Handler = (*SigCheckHandler)(nil)

func (s *SigCheckHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	s.Signers = Authenticate{}.GetConditions(ctx)
	return &weave.CheckResult{}, nil
}

func (s *SigCheckHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	s.Signers = Authenticate{}.GetConditions(ctx)
	return &weave.DeliverResult{}, nil
}

func TestGasPaymentPerSigner(t *testing.T) {
	var (
		h weavetest.Handler
		d Decorator
	)

	ctx := context.Background()
	ctx = weave.WithChainID(ctx, "mychain")
	db := store.MemStore()
	migration.MustInitPkg(db, "sigs")

	priv := weavetest.NewKey()
	tx := NewStdTx([]byte("foo"))
	if sig, err := SignTx(priv, tx, "mychain", 0); err != nil {
		t.Fatalf("cannot sign the transaction: %s", err)
	} else {
		tx.Signatures = []*StdSignature{sig}
	}

	res, err := d.Check(ctx, db, tx, &h)
	if err != nil {
		t.Fatalf("cannot check: %s", err)
	}
	if got, want := res.GasPayment, int64(signatureVerifyCost); want != got {
		t.Fatalf("want %d gas payment, got %d", want, got)
	}
}
