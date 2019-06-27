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
	kv := store.MemStore()
	migration.MustInitPkg(kv, "sigs")
	checkKv := kv.CacheWrap()
	signers := new(SigCheckHandler)
	d := NewDecorator()
	chainID := "deco-rate"
	ctx := weave.WithChainID(context.Background(), chainID)

	priv := weavetest.NewKey()
	perms := []weave.Condition{priv.PublicKey().Condition()}

	bz := []byte("art")
	tx := NewStdTx(bz)
	sig, err := SignTx(priv, tx, chainID, 0)
	assert.Nil(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	assert.Nil(t, err)

	// Order of calling first check and then deliver is important.
	cases := []struct {
		name string
		fn   func(weave.Decorator, weave.Tx) error
	}{
		{
			name: "check",
			fn: func(dec weave.Decorator, my weave.Tx) error {
				_, err := dec.Check(ctx, checkKv, my, signers)
				return err
			},
		},
		{
			name: "deliver",
			fn: func(dec weave.Decorator, my weave.Tx) error {
				_, err := dec.Deliver(ctx, kv, my, signers)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// test with no sigs
			tx.Signatures = nil
			if err := tc.fn(d, tx); !errors.ErrUnauthorized.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}

			// test with one
			tx.Signatures = []*StdSignature{sig}
			err = tc.fn(d, tx)
			assert.Nil(t, err)
			assert.Equal(t, perms, signers.Signers)

			// test with replay
			if err := tc.fn(d, tx); !ErrInvalidSequence.Is(err) {
				t.Fatalf("unexpected errror: %+v", err)
			}

			// test allowing none
			ad := d.AllowMissingSigs()
			tx.Signatures = nil
			err = tc.fn(ad, tx)
			assert.Nil(t, err)
			assert.Equal(t, []weave.Condition{}, signers.Signers)

			// test allowing, with next sequence
			tx.Signatures = []*StdSignature{sig1}
			err = tc.fn(ad, tx)
			assert.Nil(t, err)
			assert.Equal(t, perms, signers.Signers)
		})
	}

}

// SigCheckHandler stores the seen signers on each call
type SigCheckHandler struct {
	Signers []weave.Condition
}

var _ weave.Handler = (*SigCheckHandler)(nil)

func (s *SigCheckHandler) Check(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	s.Signers = Authenticate{}.GetConditions(ctx)
	return &weave.CheckResult{}, nil
}

func (s *SigCheckHandler) Deliver(ctx context.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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
