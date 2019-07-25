package cron

import (
	"strings"
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestTaskResultValidation(t *testing.T) {
	cases := map[string]struct {
		Msg *TaskResult
		// Field name to error mapping. Use `nil` if no error is expected.
		WantErrs map[string]*errors.Error
	}{
		"valid message": {
			Msg: &TaskResult{
				Metadata: &weave.Metadata{Schema: 1},
				ExecTime: weave.UnixTime(1000000),
			},
			WantErrs: map[string]*errors.Error{
				"Metadata":   nil,
				"Successful": nil,
				"Info":       nil,
				"ExecTime":   nil,
				"ExecHeight": nil,
			},
		},
		"info string too long": {
			Msg: &TaskResult{
				Metadata: &weave.Metadata{Schema: 1},
				ExecTime: weave.UnixTime(1000000),
				Info:     strings.Repeat("x", 10241),
			},
			WantErrs: map[string]*errors.Error{
				"Metadata":   nil,
				"Successful": nil,
				"Info":       errors.ErrInput,
				"ExecTime":   nil,
				"ExecHeight": nil,
			},
		},
		"missing metadata": {
			Msg: &TaskResult{
				Metadata: nil,
				ExecTime: weave.UnixTime(1000000),
			},
			WantErrs: map[string]*errors.Error{
				"Metadata":   errors.ErrMetadata,
				"Successful": nil,
				"Info":       nil,
				"ExecTime":   nil,
				"ExecHeight": nil,
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.Msg.Validate()
			for field, wantErr := range tc.WantErrs {
				assert.FieldError(t, err, field, wantErr)
			}
		})
	}
}
