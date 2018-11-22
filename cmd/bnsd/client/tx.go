package client

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/sigs"
)

// Tx is all the interfaces we need rolled into one
type Tx interface {
	weave.Tx
	sigs.SignedTx
}

// BuildSendTx will create an unsigned tx to move tokens
func BuildSendTx(src, dest weave.Address, amount x.Coin, memo string) *app.Tx {
	return &app.Tx{
		Sum: &app.Sum{&app.Sum_SendMsg{&cash.SendMsg{
			Src:    src,
			Dest:   dest,
			Amount: &amount,
			Memo:   memo,
		}},
		// TODO: add fees, etc...
		}}
}

// SignTx modifies the tx in-place, adding signatures
func SignTx(tx *app.Tx, signer *PrivateKey, chainID string, nonce int64) error {
	sig, err := sigs.SignTx(signer, tx, chainID, nonce)
	if err != nil {
		return err
	}
	tx.Signatures = append(tx.Signatures, sig)
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
