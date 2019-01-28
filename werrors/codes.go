package werrors

// Code represents the error type. Each error should have a code assigned to
// provide information of the class of issues they represent.
// Code can be used for testing what class of issue is represented by given
// error.
type Code uint32

const (
	// Zero error code is reserved for internal errors. Those are caused by
	// deep implementation issues and the details of them must not be
	// exposed outside of the application.
	// Internal error represents a failure that the client cannot fix.
	//
	// It is important that the internal error is represented by the zero
	// code - all external errors that do not provide an error code default
	// to that value.
	Internal Code = 0

	TxParse        Code = 1
	Unauthorized   Code = 2
	UnknownRequest Code = 3
	NotFound       Code = 4
	InvalidMsg     Code = 5
	InvalidModel   Code = 6
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
	case UnknownRequest:
		return "unknown request"
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

// Is returns true if given error represents an error Code equal to this one.
// This method is ment for testing error class equality (instead of instance
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
