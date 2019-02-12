package app

import (
	"fmt"
	"reflect"

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

// ExtractMsgFromSum will find a weave message from a sum if it exists
// To work, this requires sum to be a struct with one field, and that field can be cast to a weave.Msg
// Returns an error if it cannot succeed.
func ExtractMsgFromSum(sum interface{}) (weave.Msg, error) {
	// TODO: add better error messages here with new refactor
	if sum == nil {
		return nil, errors.ErrInternal("sum is <nil>")
	}
	pval := reflect.ValueOf(sum)
	if pval.Kind() != reflect.Ptr || pval.Elem().Kind() != reflect.Struct {
		return nil, errors.ErrInternal(fmt.Sprintf("invalid value: %T", sum))
	}
	val := pval.Elem()
	if val.NumField() != 1 {
		return nil, errors.ErrInternal(fmt.Sprintf("Unexpected field count: %d", val.NumField()))
	}
	field := val.Field(0).Interface()
	res, ok := field.(weave.Msg)
	if !ok {
		return nil, errors.ErrUnknownTxType(field)
	}
	return res, nil
}

// GetMsg switches over all types defined in the protobuf file
func (tx *Tx) GetMsg() (weave.Msg, error) {
	sum := tx.GetSum()
	return ExtractMsgFromSum(sum)
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
