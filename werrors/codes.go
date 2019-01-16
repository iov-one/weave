package werrors

// Code represents the error type. Each error should have a code assigned to
// provide information of the class of issues they represent.
// Error code can be used for testing
type Code uint32

const (
	// First 1000 error codes are reserved for internal errors. Those are
	// caused by deep implementation issues and the details of them must
	// not be exposed outside of the application.
	// This class of errors represents failures that the client cannot fix.
	Internal            Code = 1
	TxParse                  = 2
	Unauthorized             = 3
	UnknownRequest           = 4
	UnrecognizedAddress      = 5
	InvalidChainID           = 6

	// Codes greater than 1000 are considered public and errors with such
	// code can be fully exposed through the API. Those codes are related
	// to issues that the client can address.
	NotFound     Code = 1001
	InvalidMsg   Code = 1002
	InvalidModel Code = 1003
)

// String returns a text representation of each error code.
func (c Code) String() string {
	switch c {
	case Internal:
		return "internal"
	case TxParse:
		return "transaction parse"
	case Unauthorized:
		return "unauthorized"
	case UnrecognizedAddress:
		return "unrecognized address"
	case InvalidChainID:
		return "invalid chain ID"

	case NotFound:
		return "not found"
	case InvalidMsg:
		return "invalid message"
	case InvalidModel:
		return "invalid model"

	default:
		return "unknown"
	}
}

// Is returns true if given error contains an error Code equal to current one.
// This method is ment for testing error type equality (instead of instance
// equality).
func (c Code) Is(err error) bool {
	type coder interface {
		ABCICode() uint32
	}
	if e, ok := err.(coder); ok {
		return e.ABCICode() == uint32(c)
	}
	return false
}
