package auth

import (
	"testing"

	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
)

func TestSignBytes(t *testing.T) {
	bz := []byte("foobar")
	msg := &StdMsg{bz}
	tx := &StdTx{Msg: msg}

	// make sure the values out are sensible
	res, err := msg.Marshal()
	assert.NoError(t, err)
	assert.Equal(t, bz, res)
	assert.Equal(t, msg, tx.GetMsg())
	assert.Equal(t, bz, tx.GetSignBytes())

	// make sure sign bytes match tx
	c1 := BuildSignBytesTx(tx, "foo", 17)
	c1a := BuildSignBytes(bz, "foo", 17)
	assert.Equal(t, c1, c1a)
	assert.NotEqual(t, bz, c1)

	// make sure sign bytes change on chain_id and seq
	c2 := BuildSignBytes(bz, "food", 17)
	assert.NotEqual(t, c1, c2)
	c3 := BuildSignBytes(bz, "foo", 18)
	assert.NotEqual(t, c1, c3)
}

// func TestVerifySignatures(t *testing.T) {
// 	// set pubkey
// 	priv := crypto.GenPrivKeyEd25519()
// 	bz := []byte("my special valentine")
// 	msg := &StdMsg{bz}
// 	tx := &StdTx{Msg: msg}

//   sig0 := SignTx(priv, tx, "emo", 13)
// 	sig := SignTx(priv, tx, "emo", 13)
// }

//----- mock objects for testing...

type StdTx struct {
	Msg        *StdMsg
	Signatures []StdSignature
}

var _ SignedTx = (*StdTx)(nil)
var _ weave.Tx = (*StdTx)(nil)

func (tx StdTx) GetMsg() weave.Msg {
	return tx.Msg
}

func (tx StdTx) GetSignatures() []StdSignature {
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
