package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

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
