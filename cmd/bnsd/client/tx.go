package client

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bcpd/app"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/sigs"
)

// Tx is all the interfaces we need rolled into one
type Tx interface {
	weave.Tx
	sigs.SignedTx
	AppendSignature(sig *sigs.StdSignature)
}

type myTx struct {
	*app.Tx
}

var _ Tx = myTx{}

func (m myTx) AppendSignature(sig *sigs.StdSignature) {
	m.Tx.Signatures = append(m.Tx.Signatures, sig)
}

// BuildSendTx will create an unsigned tx to move tokens
func BuildSendTx(src, dest weave.Address, amount x.Coin, memo string) myTx {
	return myTx{&app.Tx{
		Sum: &app.Tx_SendMsg{&cash.SendMsg{
			Src:    src,
			Dest:   dest,
			Amount: &amount,
			Memo:   memo,
		}},
		// TODO: add fees, etc...
	}}
}

// BuildSetNameTx will create an unsigned tx to set a name
func BuildSetNameTx(addr weave.Address, name string) myTx {
	return myTx{&app.Tx{
		Sum: &app.Tx_SetNameMsg{&namecoin.SetWalletNameMsg{
			Address: addr,
			Name:    name,
		}},
		// TODO: add fees, etc...
	}}
}

// SignTx modifies the tx in-place, adding signatures
func SignTx(tx Tx, signer *PrivateKey, chainID string, nonce int64) error {
	sig, err := sigs.SignTx(signer, tx, chainID, nonce)
	if err != nil {
		return err
	}
	tx.AppendSignature(sig)
	return nil
}

// ParseBcpTx will load a serialize tx into a format we can read
func ParseBcpTx(data []byte) (*app.Tx, error) {
	var tx app.Tx
	err := tx.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
