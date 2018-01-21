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

func TestChecks(t *testing.T) {
	// TODO: make sure the Is and Err methods match
	assert := assert.New(t)

	cases := []struct {
		err   error
		check CheckErr
		match bool
	}{
		// specific errors match broader checks, but not visa versa
		{ErrDecoding(), IsDecodingErr, true},
		{ErrTooLarge(), IsTooLargeErr, true},
		{ErrTooLarge(), IsDecodingErr, true},
		{ErrDecoding(), IsTooLargeErr, false},

		{ErrUnauthorized(), IsDecodingErr, false},
		{ErrUnauthorized(), IsUnauthorizedErr, true},
		// make sure lots of things match InternalErr, but not everything
		{ErrInternal("bad db connection"), IsInternalErr, true},
		{Wrap(fmt.Errorf("wrapped")), IsInternalErr, true},
		{ErrUnauthorized(), IsInternalErr, false},

		{ErrMissingSignature(), IsUnauthorizedErr, true},
		{ErrMissingSignature(), IsMissingSignatureErr, true},
		{ErrUnauthorized(), IsMissingSignatureErr, false},
		{ErrInvalidSignature(), IsUnauthorizedErr, true},
		{ErrInvalidSignature(), IsInvalidSignatureErr, true},

		{nil, NoErr, true},
	}

	for i, tc := range cases {
		match := tc.check(tc.err)
		assert.Equal(tc.match, match, "%d", i)
	}
}
