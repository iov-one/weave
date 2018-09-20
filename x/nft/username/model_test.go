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
				ActionApprovals: []*nft.ActionApprovals{
					{Action: "anyActiom",
						Approvals: []*nft.Approval{
							{Address: bob.Address(),
								Options: &nft.ApprovalOptions{Count: 1},
							},
						}},
				}},
			Details: &username.TokenDetails{
				[]username.PublicKey{{Data: alice, Algorithm: "any"}},
			},
		},
		{Base: &nft.NonFungibleToken{}, Details: &username.TokenDetails{}},
		{Base: &nft.NonFungibleToken{}, Details: &username.TokenDetails{
			Keys: []username.PublicKey{}},
		},
		{Base: &nft.NonFungibleToken{ActionApprovals: []*nft.ActionApprovals{}}, Details: &username.TokenDetails{}},
		{
			Base:    &nft.NonFungibleToken{ActionApprovals: []*nft.ActionApprovals{{Approvals: []*nft.Approval{}}}},
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
	if expected.Details.Keys != nil {
		assert.Equal(t, expected.Details.Keys, actual.Details.Keys)
	} else {
		assert.Len(t, actual.Details.Keys, 0)
	}
}

func TestTokenDetailsClone(t *testing.T) {
	source := username.TokenDetails{[]username.PublicKey{{Data: []byte("foo")}, {Data: []byte("bar")}}}
	myClone := source.Clone()
	// when
	source.Keys[0].Data = source.Keys[0].Data[1:]
	source.Keys = append(source.Keys, username.PublicKey{})

	assert.NotEqual(t, source, myClone)
	assert.Len(t, myClone.Keys, 2)
	assert.Equal(t, []username.PublicKey{{Data: []byte("foo")}, {Data: []byte("bar")}}, myClone.Keys)
}
