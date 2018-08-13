package sigs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/store"
)

func TestDecorator(t *testing.T) {

	kv := store.MemStore()
	checkKv := kv.CacheWrap()
	signers := new(SigCheckHandler)
	d := NewDecorator()
	chainID := "deco-rate"
	ctx := weave.WithChainID(context.Background(), chainID)

	priv := crypto.GenPrivKeyEd25519()
	perms := []weave.Condition{priv.PublicKey().Condition()}

	bz := []byte("art")
	tx := NewStdTx(bz)
	sig, err := SignTx(priv, tx, chainID, 0)
	require.NoError(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	require.NoError(t, err)

	deliver := func(dec weave.Decorator, my weave.Tx) error {
		_, err := dec.Deliver(ctx, kv, my, signers)
		return err
	}
	check := func(dec weave.Decorator, my weave.Tx) error {
		_, err := dec.Check(ctx, checkKv, my, signers)
		return err
	}

	for i, fn := range []func(weave.Decorator, weave.Tx) error{check, deliver} {
		// test with no sigs
		tx.Signatures = nil
		err := fn(d, tx)
		assert.Error(t, err, "%d", i)

		// test with one
		tx.Signatures = []*StdSignature{sig}
		err = fn(d, tx)
		assert.NoError(t, err, "%d", i)
		assert.Equal(t, perms, signers.Signers)

		// test with replay
		err = fn(d, tx)
		assert.Error(t, err, "%d", i)

		// test allowing none
		ad := d.AllowMissingSigs()
		tx.Signatures = nil
		err = fn(ad, tx)
		assert.NoError(t, err, "%d", i)
		assert.Equal(t, []weave.Condition{}, signers.Signers)

		// test allowing, with next sequence
		tx.Signatures = []*StdSignature{sig1}
		err = fn(ad, tx)
		assert.NoError(t, err, "%d", i)
		assert.Equal(t, perms, signers.Signers)
	}

}

//---------------- helpers --------

// SigCheckHandler stores the seen signers on each call
type SigCheckHandler struct {
	Signers []weave.Condition
}

var _ weave.Handler = (*SigCheckHandler)(nil)

func (s *SigCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	s.Signers = Authenticate{}.GetConditions(ctx)
	return
}

func (s *SigCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	s.Signers = Authenticate{}.GetConditions(ctx)
	return
}
