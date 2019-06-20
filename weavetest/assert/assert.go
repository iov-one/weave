package assert

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/errors"
)

// Nil fails the test if given value is not nil.
func Nil(t testing.TB, value interface{}) {
	t.Helper()
	if !isNil(value) {
		// Use %+v so that if we are printing an error that supports
		// stack traces then a full stack trace is shown.
		t.Fatalf("want a nil value, got %+v", value)
	}
}

func isNil(value interface{}) (isnil bool) {
	if value == nil {
		return true
	}

	defer func() {
		if recover() != nil {
			isnil = false
		}
	}()

	// The argument must be a chan, func, interface, map, pointer, or slice
	// value; if it is not, IsNil panics.
	isnil = reflect.ValueOf(value).IsNil()

	return isnil
}

// Equal fails the test if two values are not equal.
func Equal(t testing.TB, want, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("values not equal \nwant %T %v\n got %T %v", want, want, got, got)
	}
}

// Panics will run given function and recover any panic. It will fail the test
// if given function call did not panic.
func Panics(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("panic expected")
		}
	}()
	fn()
}

// FieldErrors ensures that given error contains the exact match of field
// errors, tested by their type (.Is method call).
// To test that no error was found for a given field name, use `nil` as the
// match value.
func FieldErrors(t testing.TB, err error, fieldName string, mustMatch ...*errors.Error) {
	errs := errors.FieldErrors(err, fieldName)

	for _, want := range mustMatch {
		// This is a special case when we want no errors (nil).
		if want == nil && len(errs) == 0 {
			continue
		}
		if !containsError(want, errs) {
			t.Errorf("%q error not found", want)
		}
	}
}

// containsError returns true if at least one element from given collection is
// of a provided error type.
func containsError(e *errors.Error, collection []error) bool {
	for _, element := range collection {
		if e.Is(element) {
			return true
		}
	}
	return false
}
