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
	sum := tx.GetSum().GetSum()
	if sum == nil {
		return nil, errors.ErrDecoding()
	}

	// make sure to cover all messages defined in protobuf
	switch t := sum.(type) {
	case *Sum_SendMsg:
		return t.SendMsg, nil
	case *Sum_SetNameMsg:
		return t.SetNameMsg, nil
	case *Sum_NewTokenMsg:
		return t.NewTokenMsg, nil
	case *Sum_CreateEscrowMsg:
		return t.CreateEscrowMsg, nil
	case *Sum_ReleaseEscrowMsg:
		return t.ReleaseEscrowMsg, nil
	case *Sum_ReturnEscrowMsg:
		return t.ReturnEscrowMsg, nil
	case *Sum_UpdateEscrowMsg:
		return t.UpdateEscrowMsg, nil
	case *Sum_CreateContractMsg:
		return t.CreateContractMsg, nil
	case *Sum_UpdateContractMsg:
		return t.UpdateContractMsg, nil
	case *Sum_SetValidatorsMsg:
		return t.SetValidatorsMsg, nil
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
