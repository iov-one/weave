package username_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/assert"
)

func TestTokenClone(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	sources := []username.UsernameToken{
		{ // happy path
			Base: &nft.NonFungibleToken{
				Id:    []byte("anyID"),
				Owner: alice.Address(),
				ActionApprovals: []nft.ActionApprovals{
					{Action: "anyActiom",
						Approvals: []nft.Approval{
							{Address: bob.Address(),
								Options: nft.ApprovalOptions{Count: 1},
							},
						}},
				}},
			Details: &username.TokenDetails{
				[]username.ChainAddress{{ChainID: []byte("myChainID"), Address: alice.Address()}},
			},
		},
		{Base: &nft.NonFungibleToken{}, Details: &username.TokenDetails{}},
		{Base: &nft.NonFungibleToken{}, Details: &username.TokenDetails{
			Addresses: []username.ChainAddress{}},
		},
		{Base: &nft.NonFungibleToken{ActionApprovals: []nft.ActionApprovals{}}, Details: &username.TokenDetails{}},
		{
			Base:    &nft.NonFungibleToken{ActionApprovals: []nft.ActionApprovals{{Approvals: []nft.Approval{}}}},
			Details: &username.TokenDetails{},
		},
	}
	for i, source := range sources {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			myClone := source.Copy().(*username.UsernameToken)
			equals(t, source, *myClone)
		})
	}
}

func equals(t *testing.T, expected username.UsernameToken, actual username.UsernameToken) {
	assert.Equal(t, expected.Base.Id, actual.Base.Id)
	assert.Equal(t, expected.Base.Owner, actual.Base.Owner)
	if expected.Base.ActionApprovals != nil {
		assert.Equal(t, expected.Base.ActionApprovals, actual.Base.ActionApprovals)
	} else {
		assert.Len(t, actual.Base.ActionApprovals, 0)
	}
	if expected.Details.Addresses != nil {
		assert.Equal(t, expected.Details.Addresses, actual.Details.Addresses)
	} else {
		assert.Len(t, actual.Details.Addresses, 0)
	}
}

func TestTokenDetailsClone(t *testing.T) {
	source := username.TokenDetails{[]username.ChainAddress{{ChainID: []byte("foo")}, {ChainID: []byte("bar")}}}
	myClone := source.Clone()
	// when
	source.Addresses[0].ChainID = source.Addresses[0].ChainID[1:]
	source.Addresses = append(source.Addresses, username.ChainAddress{})

	assert.NotEqual(t, source, myClone)
	assert.Len(t, myClone.Addresses, 2)
	assert.Equal(t, []username.ChainAddress{{ChainID: []byte("foo")}, {ChainID: []byte("bar")}}, myClone.Addresses)
}

func TestChainAddressValidation(t *testing.T) {
	specs := []struct {
		chainID  string
		address  []byte
		expError bool
	}{
		{chainID: "1234", address: []byte("123456789012"), expError: false},
		{chainID: "1234", address: anyIDWithLength(50), expError: false},
		{chainID: "1234", address: []byte{}, expError: true}, // empty address
		{chainID: "1234", address: nil, expError: true},
		{chainID: "", address: []byte("123456789012"), expError: true},
		{chainID: "1234", address: []byte("12345678901"), expError: true},
		{chainID: "1234", address: anyIDWithLength(51), expError: true},
		{chainID: string(anyIDWithLength(257)), address: []byte("123456789012"), expError: true},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			c := username.ChainAddress{ChainID: []byte(spec.chainID), Address: []byte(spec.address)}
			if spec.expError {
				assert.Error(t, c.Validate())
			} else {
				assert.NoError(t, c.Validate())
			}
		})
	}
}
