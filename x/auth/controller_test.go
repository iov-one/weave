package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/store"
)

func TestSignBytes(t *testing.T) {
	bz := []byte("foobar")
	msg := &StdMsg{bz}
	tx := &StdTx{Msg: msg}

	bz2 := []byte("blast")
	tx2 := &StdTx{Msg: &StdMsg{bz2}}

	// make sure the values out are sensible
	res, err := msg.Marshal()
	assert.NoError(t, err)
	assert.Equal(t, bz, res)
	assert.Equal(t, msg, tx.GetMsg())
	assert.Equal(t, bz, tx.GetSignBytes())
	assert.Equal(t, bz2, tx2.GetSignBytes())

	// make sure sign bytes match tx
	c1 := BuildSignBytesTx(tx, "foo", 17)
	c1a := BuildSignBytes(bz, "foo", 17)
	assert.Equal(t, c1, c1a)
	assert.NotEqual(t, bz, c1)

	// make sure sign bytes change on tx, chain_id and seq
	ct := BuildSignBytes(bz2, "foo", 17)
	assert.NotEqual(t, c1, ct)
	c2 := BuildSignBytes(bz, "food", 17)
	assert.NotEqual(t, c1, c2)
	c3 := BuildSignBytes(bz, "foo", 18)
	assert.NotEqual(t, c1, c3)
}

func TestVerifySignature(t *testing.T) {
	kv := store.MemStore()
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	addr := pub.Address()

	chainID := "emo"
	bz := []byte("my special valentine")
	msg := &StdMsg{bz}
	tx := &StdTx{Msg: msg}

	sig0 := SignTx(priv, tx, chainID, 0)
	sig1 := SignTx(priv, tx, chainID, 1)
	sig2 := SignTx(priv, tx, chainID, 2)
	sig13 := SignTx(priv, tx, chainID, 13)
	empty := new(StdSignature)

	// signing should be deterministic
	sig2a := SignTx(priv, tx, chainID, 2)
	assert.Equal(t, sig2, sig2a)

	// the first one must have a signature in the store
	_, err := VerifySignature(kv, sig1, bz, chainID)
	assert.Error(t, err)

	// empty sig
	_, err = VerifySignature(kv, empty, bz, chainID)
	assert.Error(t, err)
	assert.True(t, IsInvalidSignatureErr(err))
	// pubkey address mismatch
	sig0x := SignTx(priv, tx, chainID, 0)
	sig0x.Address = weave.NewAddress([]byte("foo"))
	_, err = VerifySignature(kv, sig0x, bz, chainID)
	assert.Error(t, err)
	assert.True(t, IsInvalidSignatureErr(err))

	// must start with 0
	sign, err := VerifySignature(kv, sig0, bz, chainID)
	assert.NoError(t, err)
	assert.Equal(t, weave.Address(addr), sign)
	// we can advance one (store in kvstore)
	sign, err = VerifySignature(kv, sig1, bz, chainID)
	assert.NoError(t, err)
	assert.Equal(t, weave.Address(addr), sign)

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
	// doesn't match on different address in sig
	sig2.Address[0] = 42
	_, err = VerifySignature(kv, sig2, bz, chainID)
	assert.Error(t, err)
}

func TestVerifyTxSignatures(t *testing.T) {
	kv := store.MemStore()

	priv := crypto.GenPrivKeyEd25519()
	addr := weave.Address(priv.PublicKey().Address())
	priv2 := crypto.GenPrivKeyEd25519()
	addr2 := weave.Address(priv2.PublicKey().Address())

	chainID := "hot"
	bz := []byte("ice cream")
	msg := &StdMsg{bz}
	tx := &StdTx{Msg: msg}
	tx2 := &StdTx{Msg: &StdMsg{[]byte(chainID)}}
	assert.NotEqual(t, tx.GetSignBytes(), tx2.GetSignBytes())

	// two sigs from the first key
	sig := SignTx(priv, tx, chainID, 0)
	sig1 := SignTx(priv, tx, chainID, 1)
	// one from the second
	sig2 := SignTx(priv2, tx, chainID, 0)
	// and a signature of wrong info
	badSig := SignTx(priv, tx2, chainID, 0)

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
	Msg        *StdMsg
	Signatures []*StdSignature
}

var _ SignedTx = (*StdTx)(nil)
var _ weave.Tx = (*StdTx)(nil)

func (tx StdTx) GetMsg() weave.Msg {
	return tx.Msg
}

func (tx StdTx) GetSignatures() []*StdSignature {
	return tx.Signatures
}

func (tx StdTx) GetSignBytes() []byte {
	s := tx.Signatures
	tx.Signatures = nil
	// TODO: marshall self w/o sigs
	bz, _ := tx.Msg.Marshal()
	tx.Signatures = s
	return bz
}

var _ weave.Msg = (*StdMsg)(nil)

type StdMsg struct {
	data []byte
}

func (s StdMsg) Marshal() ([]byte, error) {
	return s.data, nil
}

func (s *StdMsg) Unmarshal(bz []byte) error {
	s.data = bz
	return nil
}

func (s StdMsg) Path() string {
	return "std"
}
