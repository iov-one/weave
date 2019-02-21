package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CheckErr is the type of all the check functions here
type CheckErr func(error) bool

// NoErr is useful for test cases when you want to fulfil the CheckErr type
func NoErr(err error) bool {
	return err == nil
}

// TestChecks make sure the Is and Err methods match
func TestChecks(t *testing.T) {
	cases := []struct {
		err   error
		check CheckErr
		match bool
	}{

		// make sure lots of things match ErrInternal, but not everything
		{Wrap(fmt.Errorf("wrapped"), "wrapped"), IsInternalErr, true},
		{nil, NoErr, true},
		{Wrap(nil, "asd"), NoErr, true},
	}

	for i, tc := range cases {
		match := tc.check(tc.err)
		assert.Equal(t, tc.match, match, "%d", i)
	}
}