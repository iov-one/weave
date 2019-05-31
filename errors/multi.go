package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Append combines two errors. Any value can be nil.
// When both are nil then nil is returned, too.
// When only one is not nil, that one is returned.
// Otherwise the result is of type multiErr.
func Append(src, new error) error {
	switch {
	case new == nil:
		return src
	case src == nil:
		return new
	}
	return multiErr{}.with(src).with(new)
}

const multiErrCode uint32 = 1000

var _ coder = (*multiErr)(nil)

type multiErr []error

// IsEmpty returns true if there are no errors registered,
func (e multiErr) IsEmpty() bool {
	return len(e) == 0
}

// with returns a new multiErr instance with the given error added.
// Nil values are ignored and multiErr flattened.
func (e multiErr) with(source error) multiErr {
	switch err := source.(type) {
	case nil:
		return e
	case multiErr:
		return e.append(err...)
	case *wrappedError:
		root, msgs := unWrap(err)

		me, ok := root.(multiErr)
		if !ok {
			return e.append(source)
		}
		// flatten and re-wrap errors
		result := e
		for _, v := range me {
			rErr := v
			for _, m := range msgs {
				rErr = Wrap(rErr, m)
			}
			result = append(result, rErr)
		}
		return result
	}
	return e.append(source)
}

// append copies values into a new array to not let stdlib append modify the the original one
func (e multiErr) append(errs ...error) multiErr {
	r := make(multiErr, len(e), len(e)+len(errs))
	copy(r, e)
	return append(r, errs...)
}

// Error satisfies the error interface and returns a serialized version of the content.
func (e multiErr) Error() string {
	if e.IsEmpty() {
		return ""
	}

	errs := make([]string, len(e))
	for i, err := range e {
		errs[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(e), strings.Join(errs, "\n\t"))
}

// ABCICode returns 1000
func (e multiErr) ABCICode() uint32 {
	return multiErrCode
}

// Contains returns true when the given error instance is element of this multiErr.
func (e multiErr) Contains(err *Error) bool {
	for _, v := range e {
		if err.Is(v) {
			return true
		}
	}
	return false
}

// abciLogFrame defines the frame type
type abciLogFrame struct {
	Data []abciLogElement `json:"data"`
}

// abciLogElement represents an error element of the multiErr
type abciLogElement struct {
	Code uint32 `json:"code"`
	Log  string `json:"log"`
}

// serializeMultiErr converts the given error into a json structured byte string
func serializeMultiErr(source multiErr, enc errEncoder) string {
	logs := make([]abciLogElement, len(source))
	for i, err := range source {
		code := abciCode(err)
		logs[i] = abciLogElement{Code: code, Log: enc(err)}
	}

	b, err := json.Marshal(abciLogFrame{Data: logs})
	if err != nil { // return empty but valid json
		return "{}"
	}
	return string(b)
}
