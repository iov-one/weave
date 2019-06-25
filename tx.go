package weave

import (
	"reflect"

	"github.com/iov-one/weave/errors"
)

// Msg is message for the blockchain to take an action
// (Make a state transition). It is just the request, and
// must be validated by the Handlers. All authentication
// information is in the wrapping Tx.
type Msg interface {
	Persistent

	// Return the message path.
	// This is used by the Router to locate the proper Handler. Msg should
	// be created alongside the Handler that corresponds to them.
	//
	// Multiple message types may return the same path value and will end
	// up being processed by the same Handler.
	//
	// Path value must be constructed following several rules:
	// - name must be snake_case
	// - value must be in format <extension_name>/<message_type_name> where
	//   extension_name is the same as the Go package name and the
	//   message_type_name is the snake_case converted message name.
	Path() string

	// Validate performs a sanity checks on this message. It returns an
	// error if at least one test does not pass and message is considered
	// invalid.
	// This validation performs only tests that do not require external
	// resources (ie a database).
	Validate() error
}

// Marshaller is anything that can be represented in binary
//
// Marshall may validate the data before serializing it and
// unless you previously validated the struct,
// errors should be expected.
type Marshaller interface {
	Marshal() ([]byte, error)
}

// Persistent supports Marshal and Unmarshal
//
// This is separated from Marshal, as this almost always requires
// a pointer, and functions that only need to marshal bytes can
// use the Marshaller interface to access non-pointers.
//
// As with Marshaller, this may do internal validation on the data
// and errors should be expected.
type Persistent interface {
	Marshaller
	Unmarshal([]byte) error
}

// Tx represent the data sent from the user to the chain.
// It includes the actual message, along with information needed
// to authenticate the sender (cryptographic signatures),
// and anything else needed to pass through middleware.
//
// Each Application must define their own tx type, which
// embeds all the middlewares that we wish to use.
// auth.SignedTx and token.FeeTx are common interfaces that
// many apps will wish to support.
type Tx interface {
	Persistent

	// GetMsg returns the action we wish to communicate
	GetMsg() (Msg, error)
}

// GetPath returns the path of the message, or (missing) if no message
func GetPath(tx Tx) string {
	msg, err := tx.GetMsg()
	if err == nil && msg != nil {
		return msg.Path()
	}
	return "(missing)"
}

// TxDecoder can parse bytes into a Tx
type TxDecoder func(txBytes []byte) (Tx, error)

// ExtractMsgFromSum will find a weave message from a tx sum type if it exists.
// Assuming you define your Tx with protobuf, this will help you implement GetMsg()
//
//   ExtractMsgFromSum(tx.GetSum())
//
// To work, this requires sum to be a pointer to a struct with one field,
// and that field can be cast to a Msg.
// Returns an error if it cannot succeed.
func ExtractMsgFromSum(sum interface{}) (Msg, error) {
	if sum == nil {
		return nil, errors.Wrap(errors.ErrInput, "message container is <nil>")
	}
	pval := reflect.ValueOf(sum)
	if pval.Kind() != reflect.Ptr || pval.Elem().Kind() != reflect.Struct {
		return nil, errors.Wrapf(errors.ErrInput, "invalid message container value: %T", sum)
	}
	val := pval.Elem()
	if val.NumField() != 1 {
		return nil, errors.Wrapf(errors.ErrInput, "Unexpected message container field count: %d", val.NumField())
	}
	field := val.Field(0)
	if field.IsNil() {
		return nil, errors.Wrap(errors.ErrState, "message is <nil>")
	}
	res, ok := field.Interface().(Msg)
	if !ok {
		return nil, errors.Wrapf(errors.ErrType, "unsupported message type: %T", field.Interface())
	}
	return res, nil
}

// LoadMsg extracts the message represented by given transaction into given
// destination. Before returning message validation method is called.
func LoadMsg(tx Tx, destination interface{}) error {
	msg, err := tx.GetMsg()
	if err != nil {
		return errors.Wrap(err, "cannot get transaction message")
	}
	if msg == nil {
		return errors.Wrap(errors.ErrState, "nil message")
	}

	if err := msg.Validate(); err != nil {
		return errors.Wrap(err, "invalid message")
	}

	dstVal := reflect.ValueOf(destination)
	if dstVal.Kind() != reflect.Ptr {
		return errors.Wrapf(errors.ErrType, "destination must be a pointer, got %T", destination)
	}
	dstVal = dstVal.Elem()
	if !dstVal.IsValid() {
		return errors.Wrap(errors.ErrType, "destination cannot be addressed")
	}

	srcVal := reflect.ValueOf(msg)
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}

	if srcVal.Type() != dstVal.Type() {
		return errors.Wrapf(errors.ErrType, "want %T destination, got %T", msg, destination)
	}

	dstVal.Set(srcVal)
	return nil
}
