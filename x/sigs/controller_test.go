package sigs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
)

func TestSignBytes(t *testing.T) {
	bz := []byte("foobar")
	tx := NewStdTx(bz)

	bz2 := []byte("blast")
	tx2 := NewStdTx(bz2)

	// make sure the values out are sensible
	tbz, err := tx.GetSignBytes()
	assert.NoError(t, err)
	assert.Equal(t, bz, tbz)
	tbz2, err := tx2.GetSignBytes()
	assert.NoError(t, err)
	assert.Equal(t, bz2, tbz2)

	// make sure sign bytes match tx
	chainID := "test-sign-bytes"
	c1, err := BuildSignBytesTx(tx, chainID, 17)
	require.NoError(t, err)
	c1a, err := BuildSignBytes(bz, chainID, 17)
	require.NoError(t, err)
	assert.Equal(t, c1, c1a)
	assert.NotEqual(t, bz, c1)

	// make sure sign bytes change on tx, chain_id and seq
	ct, err := BuildSignBytes(bz2, chainID, 17)
	require.NoError(t, err)
	assert.NotEqual(t, c1, ct)
	c2, err := BuildSignBytes(bz, chainID+"2", 17)
	require.NoError(t, err)
	assert.NotEqual(t, c1, c2)
	c3, err := BuildSignBytes(bz, chainID, 18)
	require.NoError(t, err)
	assert.NotEqual(t, c1, c3)
}

func TestVerifySignature(t *testing.T) {
	kv := store.MemStore()
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	perm := pub.Condition()

	chainID := "emo-music-2345"
	bz := []byte("my special valentine")
	tx := NewStdTx(bz)

	sig0, err := SignTx(priv, tx, chainID, 0)
	require.Nil(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	require.Nil(t, err)
	sig2, err := SignTx(priv, tx, chainID, 2)
	require.Nil(t, err)
	sig13, err := SignTx(priv, tx, chainID, 13)
	require.Nil(t, err)
	empty := new(StdSignature)

	// signing should be deterministic
	sig2a, err := SignTx(priv, tx, chainID, 2)
	require.Nil(t, err)
	assert.Equal(t, sig2, sig2a)

	// the first one must have a signature in the store
	_, err = VerifySignature(kv, sig1, bz, chainID)
	assert.Error(t, err)

	// empty sig
	_, err = VerifySignature(kv, empty, bz, chainID)
	assert.Error(t, err)
	assert.True(t, IsInvalidSignatureErr(err))

	// must start with 0
	sign, err := VerifySignature(kv, sig0, bz, chainID)
	assert.NoError(t, err)
	assert.Equal(t, perm, sign)
	// we can advance one (store in kvstore)
	sign, err = VerifySignature(kv, sig1, bz, chainID)
	assert.NoError(t, err)
	assert.Equal(t, perm, sign)

	// jumping and replays are a no-no
	_, err = VerifySignature(kv, sig1, bz, chainID)
	assert.Error(t, err)
	assert.True(t, IsInvalidSequenceErr(err))
	_, err = VerifySignature(kv, sig13, bz, chainID)
	assert.Error(t, err)
	assert.True(t, IsInvalidSequenceErr(err))

	// different chain doesn't match
	_, err = VerifySignature(kv, sig2, bz, "metal")
	assert.Error(t, err)
	// doesn't match on bad sig
	copy(sig2.Signature.GetEd25519(), []byte{42, 17, 99})
	_, err = VerifySignature(kv, sig2, bz, chainID)
	assert.Error(t, err)
}

func TestVerifyTxSignatures(t *testing.T) {
	kv := store.MemStore()

	priv := crypto.GenPrivKeyEd25519()
	addr := priv.PublicKey().Condition()
	priv2 := crypto.GenPrivKeyEd25519()
	addr2 := priv2.PublicKey().Condition()

	chainID := "hot_summer_days"
	bz := []byte("ice cream")
	tx := NewStdTx(bz)
	tx2 := NewStdTx([]byte(chainID))
	tbz, err := tx.GetSignBytes()
	require.NoError(t, err)
	tbz2, err := tx2.GetSignBytes()
	require.NoError(t, err)
	assert.NotEqual(t, tbz, tbz2)

	// two sigs from the first key
	sig, err := SignTx(priv, tx, chainID, 0)
	require.NoError(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	require.NoError(t, err)
	// one from the second
	sig2, err := SignTx(priv2, tx, chainID, 0)
	require.NoError(t, err)
	// and a signature of wrong info
	badSig, err := SignTx(priv, tx2, chainID, 0)
	require.NoError(t, err)

	// no signers
	signers, err := VerifyTxSignatures(kv, tx, chainID)
	assert.NoError(t, err)
	assert.Empty(t, signers)

	// bad signers
	tx.Signatures = []*StdSignature{badSig}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.Error(t, err)

	// some signers
	tx.Signatures = []*StdSignature{sig}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(signers)) {
		assert.Equal(t, addr, signers[0])
	}

	// one signature as replay is blocked
	tx.Signatures = []*StdSignature{sig, sig2}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.Error(t, err)

	// now increment seq and it passes
	tx.Signatures = []*StdSignature{sig1, sig2}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.NoError(t, err)
	if assert.Equal(t, 2, len(signers)) {
		assert.Equal(t, addr, signers[0])
		assert.Equal(t, addr2, signers[1])
	}
}

//----- mock objects for testing...

type StdTx struct {
	weave.Tx
	Signatures []*StdSignature
}

var _ SignedTx = (*StdTx)(nil)
var _ weave.Tx = (*StdTx)(nil)

func NewStdTx(payload []byte) *StdTx {
	var helpers x.TestHelpers
	msg := helpers.MockMsg(payload)
	tx := helpers.MockTx(msg)
	return &StdTx{Tx: tx}
}

func (tx StdTx) GetSignatures() []*StdSignature {
	return tx.Signatures
}

func (tx StdTx) GetSignBytes() ([]byte, error) {
	// marshal self w/o sigs
	s := tx.Signatures
	tx.Signatures = nil
	defer func() { tx.Signatures = s }()

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	bz, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	return bz, nil
}
