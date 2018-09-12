package blockchain

import (
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeliverBlockchainIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()
	auth := helpers.Authenticate(alice)

	kv := store.MemStore()
	bucket := NewBucket()

	handler := NewIssueHandler(auth, nil, bucket)
	// when
	tx := helpers.MockTx(&IssueTokenMsg{
		Owner:   alice.Address(),
		Id:      []byte("alice@example.com"),
		Details: TokenDetails{ChainID: []byte("myChainID")},
		Approvals: []*nft.Approval{{
			ToAccount: bob.Address(),
			Action:    nft.ActionKind_updatePayloadApproval,
			Options:   &nft.ApprovalOptions{Count: 1},
		}},
	})
	res, err := handler.Deliver(nil, kv, tx)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint32(0), res.ToABCI().Code)

	// and persisted
	o, err := bucket.Get(kv, []byte("alice@example.com"))
	assert.NoError(t, err)
	assert.NotNil(t, o)
	b, _ := AsBlockchainNFT(o)
	approvals := b.XApprovals(nft.ActionKind_updatePayloadApproval)
	require.Len(t, approvals, 1)
	assert.Equal(t, bob.Address(), approvals[0].ToAccountAddress())
}

func TestDeliverBlockchainUpdateMsgByNonOwner(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()
	auth := helpers.Authenticate(bob)

	kv := store.MemStore()
	bucket := NewBucket()
	o, err := bucket.Create(kv, alice.Address(), []byte("myBlockchain"), TokenDetails{
		ChainID: []byte("myChainID"),
	})
	require.NoError(t, err)
	b, _ := AsBlockchainNFT(o)
	require.NoError(t, b.SetApproval(nft.ActionKind_updatePayloadApproval, bob.Address(), &nft.ApprovalOptions{Count: 1}))
	require.NoError(t, bucket.Save(kv, o))

	handler := NewUpdateHandler(auth, nil, bucket)
	// when
	tx := helpers.MockTx(&UpdateTokenMsg{
		Id:         []byte("myBlockchain"),
		Actor:      bob.Address(),
		NewDetails: TokenDetails{ChainID: []byte("myOtherChainID")},
	})
	res, err := handler.Deliver(nil, kv, tx)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint32(0), res.ToABCI().Code)

	// and persisted
	loadedEntity, err := bucket.Get(kv, []byte("myBlockchain"))
	assert.NoError(t, err)
	assert.NotNil(t, loadedEntity)
	x, _ := AsBlockchainNFT(loadedEntity)
	assert.Equal(t, []byte("myOtherChainID"), x.Details().ChainID)
}
