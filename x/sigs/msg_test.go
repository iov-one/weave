package sigs

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestBumpSequenceValidate(t *testing.T) {
	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid message": {
			Msg: &BumpSequenceMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				Increment: 1,
				User:      weavetest.NewCondition().Address(),
			},
			WantErr: nil,
		},
		"missing user": {
			Msg: &BumpSequenceMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				Increment: 1,
			},
			WantErr: errors.ErrEmpty,
		},
		"missing metadata": {
			Msg: &BumpSequenceMsg{
				Metadata:  nil,
				Increment: 1,
				User:      weavetest.NewCondition().Address(),
			},
			WantErr: errors.ErrMetadata,
		},
		"increment too small": {
			Msg: &BumpSequenceMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				Increment: 0,
				User:      weavetest.NewCondition().Address(),
			},
			WantErr: errors.ErrMsg,
		},
		"increment too big": {
			Msg: &BumpSequenceMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				Increment: 1001,
				User:      weavetest.NewCondition().Address(),
			},
			WantErr: errors.ErrMsg,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.Msg.Validate()
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}
