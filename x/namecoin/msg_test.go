package namecoin

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateNewTokenMsg(t *testing.T) {
	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid message": {
			Msg: &NewTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "IOV",
				Name:     "foo",
				SigFigs:  5,
			},
			WantErr: nil,
		},
		"invalid ticker": {
			Msg: &NewTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "INVALID",
				Name:     "foo",
				SigFigs:  5,
			},
			WantErr: errors.ErrCurrency,
		},
		"missing metadata": {
			Msg: &NewTokenMsg{
				Ticker:  "IOV",
				Name:    "foo",
				SigFigs: 5,
			},
			WantErr: errors.ErrMetadata,
		},
		"sig figs too small": {
			Msg: &NewTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "IOV",
				Name:     "foo",
				SigFigs:  -1,
			},
			WantErr: errors.ErrInvalidInput,
		},
		"sig figs too big": {
			Msg: &NewTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "IOV",
				Name:     "foo",
				SigFigs:  10,
			},
			WantErr: errors.ErrInvalidInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}

func TestValidateSetWalletNameMsg(t *testing.T) {
	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid message": {
			Msg: &SetWalletNameMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Address:  weavetest.NewCondition().Address(),
				Name:     "foobar",
			},
			WantErr: nil,
		},
		"missing metadata": {
			Msg: &SetWalletNameMsg{
				Address: weavetest.NewCondition().Address(),
				Name:    "foobar",
			},
			WantErr: errors.ErrMetadata,
		},
		"missing address": {
			Msg: &SetWalletNameMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Address:  nil,
				Name:     "foobar",
			},
			WantErr: errors.ErrInvalidInput,
		},
		"invalid name": {
			Msg: &SetWalletNameMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Address:  weavetest.NewCondition().Address(),
				Name:     "INVALID NAME",
			},
			WantErr: errors.ErrInvalidInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}
