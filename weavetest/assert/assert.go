package assert

import (
	"reflect"
	"testing"
)

// Nil fails the test if given value is not nil.
func Nil(t testing.TB, value interface{}) {
	t.Helper()
	if !isNil(value) {
		t.Fatalf("want a nil value, got %#v", value)
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
		t.Fatalf("values not equal \nwant %v\n got %v", want, got)
	}
}
