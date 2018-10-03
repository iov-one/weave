package ticker_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	db := store.MemStore()

	bucket := ticker.NewBucket()
	blockchains := blockchain.NewBucket()
	b, _ := blockchains.Create(db, alice.Address(), []byte("alicenet"), nil)
	blockchains.Save(db, b)
	o, _ := bucket.Create(db, alice.Address(), []byte("ALC0"), []byte(string("alicenet")))
	bucket.Save(db, o)

	handler := ticker.NewIssueHandler(helpers.Authenticate(alice), nil, bucket, blockchains)

	// when
	specs := []struct {
		owner, id       []byte
		details         ticker.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner:   alice.Address(),
			id:      []byte("ALC1"),
			details: ticker.TokenDetails{[]byte("alicenet")},
		},
		// todo: add other test cases when details are specified
		// todo: add approval cases
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			tx := helpers.MockTx(&ticker.IssueTokenMsg{
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
			u, _ := ticker.AsTicker(o)
			assert.Equal(t, spec.details.BlockchainID, u.GetBlockchainID())
			// todo: verify approvals
		})
	}
}

func TestQueryTokenByName(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := ticker.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("ALC0"), []byte("myBlockchainID"))
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("BOB0"), []byte("myOtherBlockchainID"))
	bucket.Save(db, o2)

	qr := weave.NewQueryRouter()
	ticker.RegisterQuery(qr)
	// when
	h := qr.Handler("/nft/tickers")
	require.NotNil(t, h)
	mods, err := h.Query(db, "", []byte("ALC0"))
	// then
	require.NoError(t, err)
	require.Len(t, mods, 1)

	assert.Equal(t, bucket.DBKey([]byte("ALC0")), mods[0].Key)
	got, err := bucket.Parse(nil, mods[0].Value)
	require.NoError(t, err)
	x, err := ticker.AsTicker(got)
	require.NoError(t, err)
	_ = x // todo verify stored details
}
