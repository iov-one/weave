package bootstrap_node_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/bootstrap_node"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()

	bucket := bootstrap_node.NewBucket()
	blockchains := blockchain.NewBucket()
	tickerBucket := ticker.NewBucket()
	tick, err := tickerBucket.Create(db, alice.Address(), []byte("IOV"), nil, []byte("alicenet"))
	require.NoError(t, err)
	require.NoError(t, tickerBucket.Save(db, tick))
	b, err := blockchains.Create(db, alice.Address(), []byte("alicenet"), nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
	require.NoError(t, err)
	require.NoError(t, blockchains.Save(db, b))
	o, err := bucket.Create(db, alice.Address(), []byte("ALC0"), nil, []byte("alicenet"), bootstrap_node.URI{
		"ya.ru",
		10,
		"grpc",
		"",
	})
	require.NoError(t, err)
	require.NoError(t, bucket.Save(db, o))

	handler := bootstrap_node.NewIssueHandler(helpers.Authenticate(alice), nil, bucket, blockchains)

	// when
	specs := []struct {
		owner, id       []byte
		details         bootstrap_node.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner: alice.Address(),
			id:    []byte("ALC1"),
			details: bootstrap_node.TokenDetails{[]byte("alicenet"), bootstrap_node.URI{
				"ya.ru",
				10,
				"grpc",
				"",
			}},
		},
		{ // valid approvals
			owner: alice.Address(),
			id:    []byte("ALC2"),
			details: bootstrap_node.TokenDetails{[]byte("alicenet"), bootstrap_node.URI{
				"ya.ru",
				10,
				"grpc",
				"",
			}},
			approvals: []nft.ActionApprovals{{
				Action:    nft.Action_ActionUpdateDetails.String(),
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid uri
			owner: alice.Address(),
			id:    []byte("ALC2"),
			details: bootstrap_node.TokenDetails{[]byte("alicenet"), bootstrap_node.URI{
				"ya.ru",
				10,
				"grp",
				"",
			}},
			expCheckError: true,
			approvals: []nft.ActionApprovals{{
				Action:    nft.Action_ActionUpdateDetails.String(),
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid approvals
			owner: alice.Address(),
			id:    []byte("ACL3"),
			details: bootstrap_node.TokenDetails{[]byte("alicenet"), bootstrap_node.URI{
				"ya.ru",
				10,
				"grpc",
				"",
			}},
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
			tx := helpers.MockTx(&bootstrap_node.IssueTokenMsg{
				Owner:     spec.owner,
				ID:        spec.id,
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
			u, _ := bootstrap_node.AsNode(o)
			assert.Equal(t, spec.details.BlockchainID, u.GetBlockchainID())
			assert.Equal(t, spec.details.Uri, u.GetUri())

			// todo: verify approvals
		})
	}
}

func TestQueryTokenByName(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := bootstrap_node.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("ALC0"), nil, []byte("myBlockchainID"), bootstrap_node.URI{
		"ya.ru",
		10,
		"grpc",
		"",
	})
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("BOB0"), nil, []byte("myOtherBlockchainID"), bootstrap_node.URI{
		"ya.ru",
		10,
		"grpc",
		"",
	})
	bucket.Save(db, o2)

	qr := weave.NewQueryRouter()
	bootstrap_node.RegisterQuery(qr)
	// when
	h := qr.Handler("/nft/bootstrap_nodes")
	require.NotNil(t, h)
	mods, err := h.Query(db, "", []byte("ALC0"))
	// then
	require.NoError(t, err)
	require.Len(t, mods, 1)

	assert.Equal(t, bucket.DBKey([]byte("ALC0")), mods[0].Key)
	got, err := bucket.Parse(nil, mods[0].Value)
	require.NoError(t, err)
	x, err := bootstrap_node.AsNode(got)
	require.NoError(t, err)
	_ = x // todo verify stored details
}
