package errors

import (
	"fmt"
	"strings"
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
		{fmt.Errorf("wrapped"), IsInternalErr, true},
		{ErrUnauthorized(), IsInternalErr, false},

		{ErrMissingSignature(), IsUnauthorizedErr, true},
		{ErrMissingSignature(), IsMissingSignatureErr, true},
		{ErrUnauthorized(), IsMissingSignatureErr, false},
		{ErrInvalidSignature(), IsUnauthorizedErr, true},
		{ErrInvalidSignature(), IsInvalidSignatureErr, true},

		{nil, NoErr, true},
		{Wrap(nil), NoErr, true},
	}

	for i, tc := range cases {
		match := tc.check(tc.err)
		assert.Equal(t, tc.match, match, "%d", i)
	}
}

// TestLog checks the text returned by the error
func TestLog(t *testing.T) {
	cases := []struct {
		err error
		// this should always pass, just to verify
		check CheckErr
		// this is the text we want to see with .Log()
		log string
	}{
		// make sure messages are nice, even if wrapped or not
		{ErrTooLarge(), IsTooLargeErr, "(2) Input size too large"},
		{Wrap(ErrTooLarge()), IsTooLargeErr, "(2) Input size too large"},
		{Wrap(fmt.Errorf("wrapped")), IsInternalErr, "(1) wrapped"},

		// with code shouldn't change the error message
		{WithCode(ErrUnauthorized(), CodeTxParseError), IsDecodingErr, "(2) Unauthorized"},

		// with log should add some in front
		{WithLog("Special", ErrUnauthorized(), CodeInternalErr), IsInternalErr, "(1) Special: Unauthorized"},

		// verify some standard message types with prefixes
		{ErrUnrecognizedAddress([]byte{0, 0x12, 0x77}), IsUnrecognizedAddressErr, "(5) 001277: Unrecognized Address"},
		{ErrUnrecognizedCondition([]byte{0xF0, 0x0D, 0xCA, 0xFE}), IsUnrecognizedConditionErr, "(5) F00DCAFE: Unrecognized Condition"},
		{ErrUnknownTxType("john_123"), IsUnknownTxTypeErr, "(4) string: Tx type unknown"},
		{ErrUnknownTxType(t), IsUnknownTxTypeErr, "(4) *testing.T: Tx type unknown"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {

			assert.True(t, tc.check(tc.err))

			// make sure we have a nice error message with code
			msg := fmt.Sprintf("%s", tc.err)
			assert.Equal(t, tc.log, msg)

			// make sure we have a nice error message with code
			middle := fmt.Sprintf("%v", tc.err)
			assert.Contains(t, middle, tc.log)
			assert.Contains(t, middle, "common_test.go")

			// make sure we also get stack dumps....
			stack := fmt.Sprintf("%+v", tc.err)
			// we should trim off unneeded stuff
			withCode := "github.com/confio/weave/errors.WithCode\n"
			thisTest := "github.com/confio/weave/errors.TestLog\n"
			assert.False(t, strings.Contains(stack, withCode))
			assert.True(t, strings.Contains(stack, thisTest))
		})
	}
}
