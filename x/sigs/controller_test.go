package sigs

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSignBytes(t *testing.T) {
	bz := []byte("foobar")
	tx := NewStdTx(bz)

	bz2 := []byte("blast")
	tx2 := NewStdTx(bz2)

	// make sure the values out are sensible
	tbz, err := tx.GetSignBytes()
	assert.Nil(t, err)
	assert.Equal(t, bz, tbz)
	tbz2, err := tx2.GetSignBytes()
	assert.Nil(t, err)
	assert.Equal(t, bz2, tbz2)

	// make sure sign bytes match tx
	chainID := "test-sign-bytes"
	c1, err := BuildSignBytesTx(tx, chainID, 17)
	assert.Nil(t, err)
	c1a, err := BuildSignBytes(bz, chainID, 17)
	assert.Nil(t, err)
	assert.Equal(t, c1, c1a)
	if bytes.Equal(bz, c1) {
		t.Fatal("")
	}

	// make sure sign bytes change on tx, chain_id and seq
	ct, err := BuildSignBytes(bz2, chainID, 17)
	assert.Nil(t, err)
	if bytes.Equal(c1, ct) {
		t.Fatal("signature reproduced")
	}
	c2, err := BuildSignBytes(bz, chainID+"2", 17)
	assert.Nil(t, err)
	if bytes.Equal(c1, c2) {
		t.Fatal("signature reproduced")
	}
	c3, err := BuildSignBytes(bz, chainID, 18)
	assert.Nil(t, err)
	if bytes.Equal(c1, c3) {
		t.Fatal("signature reproduced")
	}
}

func TestVerifySignature(t *testing.T) {
	kv := store.MemStore()
	migration.MustInitPkg(kv, "sigs")
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	perm := pub.Condition()

	chainID := "emo-music-2345"
	bz := []byte("my special valentine")
	tx := NewStdTx(bz)

	sig0, err := SignTx(priv, tx, chainID, 0)
	assert.Nil(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	assert.Nil(t, err)
	sig2, err := SignTx(priv, tx, chainID, 2)
	assert.Nil(t, err)
	sig13, err := SignTx(priv, tx, chainID, 13)
	assert.Nil(t, err)
	empty := new(StdSignature)

	// signing should be deterministic
	sig2a, err := SignTx(priv, tx, chainID, 2)
	assert.Nil(t, err)
	assert.Equal(t, sig2, sig2a)

	// the first one must have a signature in the store
	if _, err := VerifySignature(kv, sig1, bz, chainID); !ErrInvalidSequence.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// empty sig
	if _, err := VerifySignature(kv, empty, bz, chainID); !errors.ErrUnauthorized.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// must start with 0
	sign, err := VerifySignature(kv, sig0, bz, chainID)
	assert.Nil(t, err)
	assert.Equal(t, perm, sign)

	// we can advance one (store in kvstore)
	sign, err = VerifySignature(kv, sig1, bz, chainID)
	assert.Nil(t, err)
	assert.Equal(t, perm, sign)

	// jumping and replays are a no-no
	if _, err := VerifySignature(kv, sig1, bz, chainID); !ErrInvalidSequence.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, err := VerifySignature(kv, sig13, bz, chainID); !ErrInvalidSequence.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// different chain doesn't match
	if _, err := VerifySignature(kv, sig2, bz, "metal"); !errors.ErrInput.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// doesn't match on bad sig
	copy(sig2.Signature.GetEd25519(), []byte{42, 17, 99})
	if _, err := VerifySignature(kv, sig2, bz, chainID); !errors.ErrUnauthorized.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestVerifyTxSignatures(t *testing.T) {
	kv := store.MemStore()
	migration.MustInitPkg(kv, "sigs")

	priv := weavetest.NewKey()
	addr := priv.PublicKey().Condition()
	priv2 := weavetest.NewKey()
	addr2 := priv2.PublicKey().Condition()

	chainID := "hot_summer_days"
	bz := []byte("ice cream")
	tx := NewStdTx(bz)
	tx2 := NewStdTx([]byte(chainID))
	tbz, err := tx.GetSignBytes()
	assert.Nil(t, err)
	tbz2, err := tx2.GetSignBytes()
	assert.Nil(t, err)
	if bytes.Equal(tbz, tbz2) {
		t.Fatal("signature repeated")
	}

	// two sigs from the first key
	sig, err := SignTx(priv, tx, chainID, 0)
	assert.Nil(t, err)
	sig1, err := SignTx(priv, tx, chainID, 1)
	assert.Nil(t, err)
	// one from the second
	sig2, err := SignTx(priv2, tx, chainID, 0)
	assert.Nil(t, err)
	// and a signature of wrong info
	badSig, err := SignTx(priv, tx2, chainID, 0)
	assert.Nil(t, err)

	// no signers
	signers, err := VerifyTxSignatures(kv, tx, chainID)
	assert.Nil(t, err)
	assert.Equal(t, len(signers), 0)

	// bad signers
	tx.Signatures = []*StdSignature{badSig}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	if !errors.ErrUnauthorized.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// some signers
	tx.Signatures = []*StdSignature{sig}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.Nil(t, err)
	assert.Equal(t, []weave.Condition{addr}, signers)

	// one signature as replay is blocked
	tx.Signatures = []*StdSignature{sig, sig2}
	if _, err := VerifyTxSignatures(kv, tx, chainID); !ErrInvalidSequence.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	// now increment seq and it passes
	tx.Signatures = []*StdSignature{sig1, sig2}
	signers, err = VerifyTxSignatures(kv, tx, chainID)
	assert.Nil(t, err)
	assert.Equal(t, []weave.Condition{addr, addr2}, signers)
}

type StdTx struct {
	weave.Tx
	Signatures []*StdSignature
}

var _ SignedTx = (*StdTx)(nil)
var _ weave.Tx = (*StdTx)(nil)

func NewStdTx(payload []byte) *StdTx {
	return &StdTx{
		Tx: &weavetest.Tx{
			Msg: &weavetest.Msg{Serialized: payload},
		},
	}
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
