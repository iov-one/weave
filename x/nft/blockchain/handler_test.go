package blockchain_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := blockchain.NewBucket()
	o, _ := bucket.Create(db, bob.Address(), []byte("any_network"), nil, blockchain.Chain{}, blockchain.IOV{})
	bucket.Save(db, o)

	handler := blockchain.NewIssueHandler(helpers.Authenticate(alice), nil, bucket)

	// when
	specs := []struct {
		owner, id       []byte
		details         blockchain.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner:   alice.Address(),
			id:      []byte("other_netowork"),
			details: blockchain.TokenDetails{Chain: blockchain.Chain{}, Iov: blockchain.IOV{}},
		},
		{ // valid approvals
			owner:   alice.Address(),
			id:      []byte("other_netowork1"),
			details: blockchain.TokenDetails{Chain: blockchain.Chain{}, Iov: blockchain.IOV{}},
			approvals: []nft.ActionApprovals{{
				Action:    nft.Action_ActionUpdateDetails.String(),
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid approvals
			owner:           alice.Address(),
			id:              []byte("other_netowork2"),
			details:         blockchain.TokenDetails{Chain: blockchain.Chain{}, Iov: blockchain.IOV{}},
			expCheckError:   true,
			expDeliverError: true,
			approvals: []nft.ActionApprovals{{
				Action:    "12",
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{}, Address: nil}},
			}},
		},
		// todo: add other test cases when details are specified
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			tx := helpers.MockTx(&blockchain.IssueTokenMsg{
				Owner:     spec.owner,
				Id:        spec.id,
				Details:   spec.details,
				Approvals: spec.approvals,
			})

			// when
			cache := db.CacheWrap()
			_, err := handler.Check(nil, cache, tx)
			cache.Discard()
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			// then
			require.NoError(t, err)

			// and when delivered
			res, err := handler.Deliver(nil, db, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err := bucket.Get(db, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := blockchain.AsBlockchain(o)
			assert.Equal(t, spec.details.Chain, u.GetChain())
			assert.Equal(t, spec.details.Iov, u.GetIov())
			// todo: verify approvals
		})
	}
}

func TestQueryTokenByName(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := blockchain.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("alicenet"), nil, blockchain.Chain{}, blockchain.IOV{})
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("bobnet"), nil, blockchain.Chain{}, blockchain.IOV{})
	bucket.Save(db, o2)

	qr := weave.NewQueryRouter()
	blockchain.RegisterQuery(qr)
	// when
	h := qr.Handler("/nft/blockchains")
	require.NotNil(t, h)
	mods, err := h.Query(db, "", []byte("alicenet"))
	// then
	require.NoError(t, err)
	require.Len(t, mods, 1)

	assert.Equal(t, bucket.DBKey([]byte("alicenet")), mods[0].Key)
	got, err := bucket.Parse(nil, mods[0].Value)
	require.NoError(t, err)
	x, err := blockchain.AsBlockchain(got)
	require.NoError(t, err)
	_ = x // todo verify stored details
}
