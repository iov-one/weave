package errors

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestAddToMulitiErr(t *testing.T) {
	specs := map[string]struct {
		src multiErr
		add []error
		exp multiErr
	}{
		"Add single error": {src: MultiErr, add: []error{ErrNotFound}, exp: multiErr{ErrNotFound}},
		"Add multiple":     {src: MultiErr, add: []error{ErrNotFound, ErrMsg}, exp: multiErr{ErrNotFound, ErrMsg}},
		"Add multiErr should be flattened": {
			src: MultiErr, add: []error{MultiErr.With(ErrNotFound).With(ErrMsg)}, exp: multiErr{ErrNotFound, ErrMsg},
		},
		"Add empty multiErr should be skipped": {
			src: MultiErr, add: []error{MultiErr}, exp: multiErr{},
		},
		"Add duplicates":            {src: MultiErr, add: []error{ErrNotFound, ErrNotFound}, exp: multiErr{ErrNotFound, ErrNotFound}},
		"Add nothing":               {src: MultiErr, exp: multiErr{}},
		"Add nil should be skipped": {src: MultiErr, add: []error{nil}, exp: multiErr{}},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			me := spec.src
			for _, v := range spec.add {
				me = me.With(v)
			}
			// then
			if exp, got := spec.exp, me; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

		})
	}
}

func TestMulitiErrIsEmpty(t *testing.T) {
	specs := map[string]struct {
		src multiErr
		exp bool
	}{
		"Single error": {src: MultiErr.With(ErrNotFound), exp: false},
		"Empty":        {src: MultiErr, exp: true},
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
	if exp, got := uint32(100), MultiErr.ABCICode(); exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
}

func TestMulitiErrABCICodeRegisterd(t *testing.T) {
	assert.Panics(t, func() {
		Register(multiErrCode, "fails")
	})
}
