package weave

// Msg is message for the blockchain to take an action
// (Make a state transition). It is just the request, and
// must be validated by the Handlers. All authentication
// information is in the wrapping Tx.
type Msg interface {
	Persistent

	// Return the message path.
	// This is used by the Router to locate the proper Handler.
	// Msg should be created alongside the Handler that corresponds to them.
	//
	// Multiple types may have the same value, and will end up at the
	// same Handler.
	//
	// Must be alphanumeric [0-9A-Za-z_\-]+
	Path() string
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
