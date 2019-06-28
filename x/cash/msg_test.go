package cash

import (
	"strings"
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateSendMsg(t *testing.T) {
	addr1 := weavetest.NewCondition().Address()
	addr2 := weavetest.NewCondition().Address()

	cases := map[string]struct {
		msg     weave.Msg
		wantErr *errors.Error
	}{
		"success": {
			msg: &SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Amount:      coin.NewCoinp(10, 0, "FOO"),
				Destination: addr1,
				Source:      addr2,
				Memo:        "some memo message",
				Ref:         []byte("some reference"),
			},
			wantErr: nil,
		},
		"success with minimal amount of data": {
			msg: &SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Amount:      coin.NewCoinp(10, 0, "FOO"),
				Destination: addr1,
				Source:      addr2,
			},
			wantErr: nil,
		},
		"empty message": {
			msg: &SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			wantErr: errors.ErrAmount,
		},
		"missing source": {
			msg: &SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Amount:      coin.NewCoinp(10, 0, "FOO"),
				Destination: addr1,
			},
			wantErr: errors.ErrEmpty,
		},
		"missing destination": {
			msg: &SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   coin.NewCoinp(10, 0, "FOO"),
				Source:   addr2,
			},
			wantErr: errors.ErrEmpty,
		},
		"reference too long": {
			msg: &SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Amount:      coin.NewCoinp(10, 0, "FOO"),
				Destination: addr1,
				Source:      addr2,
				Ref:         []byte(strings.Repeat("x", maxRefSize+1)),
			},
			wantErr: errors.ErrState,
		},
		"memo too long": {
			msg: &SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Amount:      coin.NewCoinp(10, 0, "FOO"),
				Destination: addr1,
				Source:      addr2,
				Memo:        strings.Repeat("x", maxMemoSize+1),
			},
			wantErr: errors.ErrState,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.msg.Validate(); !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}

func TestValidateFeeTx(t *testing.T) {
	addr1 := weavetest.NewCondition().Address()

	cases := map[string]struct {
		info    *FeeInfo
		wantErr *errors.Error
	}{
		"success": {
			info: &FeeInfo{
				Fees:  coin.NewCoinp(1, 0, "IOV"),
				Payer: addr1,
			},
			wantErr: nil,
		},
		"empty": {
			info:    &FeeInfo{},
			wantErr: errors.ErrAmount,
		},
		"no fee": {
			info: &FeeInfo{
				Payer: addr1,
			},
			wantErr: errors.ErrAmount,
		},
		"no payer": {
			info: &FeeInfo{
				Fees: coin.NewCoinp(10, 0, "IOV"),
			},
			wantErr: errors.ErrEmpty,
		},
		"negative fee": {
			info: &FeeInfo{
				Fees:  coin.NewCoinp(-10, 0, "IOV"),
				Payer: addr1,
			},
			wantErr: errors.ErrAmount,
		},
		"invalid fee ticker": {
			info: &FeeInfo{
				Fees:  coin.NewCoinp(10, 0, "foobar"),
				Payer: addr1,
			},
			wantErr: errors.ErrCurrency,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.info.Validate(); !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}
