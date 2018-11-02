package username_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/assert"
)

func TestIssueTokenMsgValidate(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	specs := []struct {
		msg      username.IssueTokenMsg
		expError bool
	}{
		{ // happy path email
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("alice@example.com"),
			},
			expError: false,
		},
		{ // happy path twitter
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("@iov_official"),
			},
			expError: false,
		},
		{ // happy path phone
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("+491234567890"),
			},
			expError: false,
		},
		{ // other characters
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("+-,._@"),
			},
			expError: false,
		},
		{ // owner missing
			msg: username.IssueTokenMsg{
				Id: []byte("alice@example.com"),
			},
			expError: true,
		},
		{ // owner wrong format
			msg: username.IssueTokenMsg{
				Owner: []byte("not an address"),
				Id:    []byte("alice@example.com"),
			},
			expError: true,
		},
		{ // id too short
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("foo"),
			},
			expError: true,
		},
		{ // id too long
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    anyIDWithLength(65),
			},
			expError: true,
		},
		{ // id with forbidden character *
			msg: username.IssueTokenMsg{
				Owner: alice.Address(),
				Id:    []byte("foo*bar"),
			},
			expError: true,
		},
		// TODO: Add checks for approvals
		// TODO: Add checks for TokenDetails
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.msg.Validate()
			if spec.expError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddChainAddressMsgValidate(t *testing.T) {
	specs := []struct {
		msg      username.AddChainAddressMsg
		expError bool
	}{
		{ // happy path
			msg: username.AddChainAddressMsg{
				Id:        []byte("me@example.com"),
				Addresses: &username.ChainAddress{[]byte("myChain"), []byte("myChainAddress")},
			},
		}, { // address missing
			msg: username.AddChainAddressMsg{
				Id:        []byte("me@example.com"),
				Addresses: &username.ChainAddress{[]byte("myChain"), nil},
			},
			expError: true,
		}, { // id missing
			msg: username.AddChainAddressMsg{
				Addresses: &username.ChainAddress{[]byte("myChain"), []byte("myChainAddress")},
			},
			expError: true,
		},
		{ // chainID missing
			msg: username.AddChainAddressMsg{
				Addresses: &username.ChainAddress{[]byte("example.com"), []byte("myChainAddress")},
			},
			expError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.msg.Validate()
			if spec.expError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestRemoveChainAddressMsgValidate(t *testing.T) {
	specs := []struct {
		msg      username.RemoveChainAddressMsg
		expError bool
	}{
		{ // happy path
			msg: username.RemoveChainAddressMsg{
				Id:        []byte("me@example.com"),
				Addresses: &username.ChainAddress{[]byte("myChain"), []byte("myChainAddress")},
			},
		}, { // address missing
			msg: username.RemoveChainAddressMsg{
				Id:        []byte("me@example.com"),
				Addresses: &username.ChainAddress{[]byte("myChain"), nil},
			},
			expError: true,
		}, { // id missing
			msg: username.RemoveChainAddressMsg{
				Addresses: &username.ChainAddress{[]byte("myChain"), []byte("myChainAddress")},
			},
			expError: true,
		},
		{ // chainID missing
			msg: username.RemoveChainAddressMsg{
				Addresses: &username.ChainAddress{[]byte("me@example.com"), []byte("myChainAddress")},
			},
			expError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.msg.Validate()
			if spec.expError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func anyIDWithLength(n int) []byte {
	r := make([]byte, n)
	for i := 0; i < n; i++ {
		r[i] = byte('a')
	}
	return r
}
