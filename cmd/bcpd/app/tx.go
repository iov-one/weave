package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/hashlock"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/sigs"
)

//-------------------------------
// copied from weave/app verbatim
//
// any cleaner way to extend a tx with functionality?

// TxDecoder creates a Tx and unmarshals bytes into it
func TxDecoder(bz []byte) (weave.Tx, error) {
	tx := new(Tx)
	err := tx.Unmarshal(bz)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// make sure tx fulfills all interfaces
var _ weave.Tx = (*Tx)(nil)
var _ cash.FeeTx = (*Tx)(nil)
var _ sigs.SignedTx = (*Tx)(nil)
var _ hashlock.HashKeyTx = (*Tx)(nil)
var _ multisig.MultiSigTx = (*Tx)(nil)

// GetMsg switches over all types defined in the protobuf file
func (tx *Tx) GetMsg() (weave.Msg, error) {
	sum := tx.GetSum()
	if sum == nil {
		return nil, errors.ErrDecoding()
	}

	// make sure to cover all messages defined in protobuf
	switch t := sum.(type) {
	case *Tx_SendMsg:
		return t.SendMsg, nil
	case *Tx_SetNameMsg:
		return t.SetNameMsg, nil
	case *Tx_NewTokenMsg:
		return t.NewTokenMsg, nil
	case *Tx_CreateEscrowMsg:
		return t.CreateEscrowMsg, nil
	case *Tx_ReleaseEscrowMsg:
		return t.ReleaseEscrowMsg, nil
	case *Tx_ReturnEscrowMsg:
		return t.ReturnEscrowMsg, nil
	case *Tx_UpdateEscrowMsg:
		return t.UpdateEscrowMsg, nil
	case *Tx_CreateContractMsg:
		return t.CreateContractMsg, nil
	case *Tx_UpdateContractMsg:
		return t.UpdateContractMsg, nil
	case *Tx_SetValidatorsMsg:
		return t.SetValidatorsMsg, nil
	case *Tx_BatchMsg:
		return t.BatchMsg, nil
	}

	// we must have covered it above
	return nil, errors.ErrUnknownTxType(sum)
}

// GetSignBytes returns the bytes to sign...
func (tx *Tx) GetSignBytes() ([]byte, error) {
	// temporarily unset the signatures, as the sign bytes
	// should only come from the data itself, not previous signatures
	sigs := tx.Signatures
	tx.Signatures = nil

	bz, err := tx.Marshal()

	// reset the signatures after calculating the bytes
	tx.Signatures = sigs
	return bz, err
}
