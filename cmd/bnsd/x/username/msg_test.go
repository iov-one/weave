package username

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestRegisterUsernameTokenMsgValidate(t *testing.T) {
	cases := map[string]struct {
		Msg  weave.Msg
		Want *errors.Error
	}{
		"valid message": {
			Msg: &RegisterUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				Targets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("1234567890")},
				},
			},
			Want: nil,
		},
		"invalid username": {
			Msg: &RegisterUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "xxx",
				Targets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("1234567890")},
				},
			},
			Want: errors.ErrInput,
		},
		"missing target": {
			Msg: &RegisterUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				Targets:  nil,
			},
			Want: errors.ErrEmpty,
		},
		"different address but the same blockchain ID is not allowed": {
			Msg: &RegisterUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				Targets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("a-blobchain-id-1")},
					{BlockchainID: "blobchain", Address: []byte("a-blobchain-id-2")},
				},
			},
			Want: errors.ErrDuplicate,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.Want.Is(err) {
				t.Fatal(err)
			}
		})
	}
}

func TestTransferUsernameTokenMsgValidate(t *testing.T) {
	cases := map[string]struct {
		Msg  weave.Msg
		Want *errors.Error
	}{
		"valid message": {
			Msg: &TransferUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				NewOwner: weavetest.NewCondition().Address(),
			},
			Want: nil,
		},
		"invalid new owner address": {
			Msg: &TransferUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				NewOwner: []byte("x"),
			},
			Want: errors.ErrInput,
		},
		"invalid username": {
			Msg: &TransferUsernameTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "xx",
				NewOwner: weavetest.NewCondition().Address(),
			},
			Want: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.Want.Is(err) {
				t.Fatal(err)
			}
		})
	}
}

func TestChangeUsernameTokenTargetsMsgValidate(t *testing.T) {
	cases := map[string]struct {
		Msg  weave.Msg
		Want *errors.Error
	}{
		"valid message": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				NewTargets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("1234567890")},
				},
			},
			Want: nil,
		},
		"invalid new targets": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				NewTargets: []BlockchainAddress{
					{BlockchainID: "x", Address: []byte("x")},
				},
			},
			Want: errors.ErrInput,
		},
		"missing new targets": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				Username:   "alice*iov",
				NewTargets: []BlockchainAddress{},
			},
			Want: errors.ErrEmpty,
		},
		"invalid username": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "xx",
				NewTargets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("1234567890")},
				},
			},
			Want: errors.ErrInput,
		},
		"invalid username separator": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice@iov",
				NewTargets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("1234567890")},
				},
			},
			Want: errors.ErrInput,
		},
		"different address but the same blockchain ID is not allowed": {
			Msg: &ChangeUsernameTokenTargetsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Username: "alice*iov",
				NewTargets: []BlockchainAddress{
					{BlockchainID: "blobchain", Address: []byte("a-blobchain-id-1")},
					{BlockchainID: "blobchain", Address: []byte("a-blobchain-id-2")},
				},
			},
			Want: errors.ErrDuplicate,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.Want.Is(err) {
				t.Fatal(err)
			}
		})
	}
}