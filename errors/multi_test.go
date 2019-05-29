package errors

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
	"github.com/pkg/errors"
)

func TestAddToMulitiErr(t *testing.T) {
	var (
		// create errors with stacktrace for equal comparision
		myErrNotFound = errors.WithStack(ErrNotFound)
		myErrState    = errors.WithStack(ErrState)
		myErrMsg      = errors.WithStack(ErrMsg)
	)
	specs := map[string]struct {
		src error
		add error
		exp error
	}{
		"Append with first nil":    {src: nil, add: myErrNotFound, exp: myErrNotFound},
		"Append with second nil":   {src: myErrNotFound, add: nil, exp: myErrNotFound},
		"Append with both nil":     {src: nil, add: nil, exp: nil},
		"Append with both not nil": {src: myErrNotFound, add: myErrMsg, exp: multiErr{myErrNotFound, myErrMsg}},
		"Append multiErr should be flattened": {
			src: myErrNotFound, add: Append(myErrState, myErrMsg), exp: multiErr{myErrNotFound, myErrState, myErrMsg},
		},
		"Append first wrapped multiErr should be flattened": {
			src: Wrap(Append(myErrState, myErrMsg), "test"),
			add: ErrHuman,
			exp: multiErr{Wrap(myErrState, "test"), Wrap(myErrMsg, "test"), ErrHuman},
		},
		"Append second wrapped multiErr should be flattened": {
			src: myErrNotFound,
			add: Wrap(Append(myErrState, myErrMsg), "test"),
			exp: multiErr{myErrNotFound, Wrap(myErrState, "test"), Wrap(myErrMsg, "test")},
		},
		"Append double wrapped multiErr should be flattened": {
			src: myErrNotFound,
			add: Wrap(Wrap(Append(myErrState, myErrMsg), "first"), "second"),
			exp: multiErr{
				myErrNotFound,
				Wrap(Wrap(myErrState, "first"), "second"),
				Wrap(Wrap(myErrMsg, "first"), "second"),
			},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			mErr := Append(spec.src, spec.add)
			// then
			if exp, got := spec.exp, mErr; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %#v but got %#v", exp, got)
			}
		})
	}
}

func TestMulitiErrIsEmpty(t *testing.T) {
	specs := map[string]struct {
		src multiErr
		exp bool
	}{
		"Single error": {src: multiErr{ErrNotFound}, exp: false},
		"Empty":        {src: multiErr{}, exp: true},
		"nil":          {src: nil, exp: true},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			// then
			if exp, got := spec.exp, spec.src.IsEmpty(); !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

		})
	}
}

func TestMulitiErrABCICode(t *testing.T) {
	var mErr multiErr
	if exp, got := uint32(1000), mErr.ABCICode(); exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
}

func TestMulitiErrABCICodeRegisterd(t *testing.T) {
	assert.Panics(t, func() {
		Register(multiErrCode, "fails")
	})
}
