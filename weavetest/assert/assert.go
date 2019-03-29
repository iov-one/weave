package assert

import (
	"reflect"
	"testing"
)

// NoErr fails the test if given error is not nil.
func NoErr(t testing.TB, err error) {
	t.Helper()
	if !errIsNil(err) {
		t.Fatalf("want the error to be nil, got %+v", err)
	}
}

// errIsNil returns true if value represented by the given error is nil.
//
// Most of the time a simple == check is enough. There is a very narrowed
// spectrum of cases  where a more sophisticated check is required.
func errIsNil(err error) bool {
	if err == nil {
		return true
	}
	if val := reflect.ValueOf(err); val.Kind() == reflect.Ptr {
		return val.IsNil()
	}
	return false
}

// Equal fails the test if two values are not equal.
func Equal(t testing.TB, want, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("values not equal \nwant %v\n got %v", want, got)
	}
}
