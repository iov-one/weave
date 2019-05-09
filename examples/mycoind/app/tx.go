package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/sigs"
)

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

// GetMsg switches over all types defined in the protobuf file
func (tx *Tx) GetMsg() (weave.Msg, error) {
	sum := tx.GetSum()
	if sum == nil {
		return nil, errors.Wrap(errors.ErrInput, "unable to decode")
	}

	// make sure to cover all messages defined in protobuf
	switch t := sum.(type) {
	case *Tx_SendMsg:
		return t.SendMsg, nil
	case *Tx_SetValidatorsMsg:
		return t.SetValidatorsMsg, nil
	}

	// we must have covered it above
	panic(sum)
	// return nil, errors.ErrUnknownTxType(nil) // alpe????
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
