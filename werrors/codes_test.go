package werrors

import (
	"errors"
	"testing"
)

func TestCodeEquality(t *testing.T) {
	cases := map[string]struct {
		err  error
		code Code
	}{
		"custom weave error": {
			err:  New(NotFound, "404"),
			code: NotFound,
		},
		"custom weave error wrapped": {
			err:  Wrap(New(NotFound, "404"), "not found"),
			code: NotFound,
		},
		"internal error is internal": {
			err:  errors.New("internal stdlib failure"),
			code: Internal,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if !tc.code.Is(tc.err) {
				t.Fatal("unexpected Is result")
			}
		})
	}
}

func TestCodeIs(t *testing.T) {
	for code := Code(0); code < 15; code++ {
		if !code.Is(codedError(code)) {
			t.Errorf("comparing to itself fails")
		}
		if code.Is(codedError(code + 1)) {
			t.Errorf("comparing to higher code must fail")
		}
		if code.Is(codedError(code - 1)) {
			t.Errorf("comparing to lower code must fail")
		}
	}
}

type codedError Code

func (e codedError) Error() string    { return "yolo" }
func (e codedError) ABCICode() uint32 { return uint32(e) }
