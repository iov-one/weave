package username_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()
	myPubKey := username.PublicKey{Data: []byte("my Public key"), Algorithm: "myAlgorithm"}

	db := store.MemStore()
	bucket := username.NewBucket()
	o, _ := bucket.Create(db, bob.Address(), []byte("existing@example.com"), []username.PublicKey{myPubKey})
	bucket.Save(db, o)
	o, err := bucket.Get(db, []byte("existing@example.com"))
	require.NoError(t, err)
	require.NotNil(t, o)

	handler := username.NewIssueHandler(helpers.Authenticate(alice), nil, bucket)

	// when
	specs := []struct {
		owner, id       []byte
		details         username.TokenDetails
		approvals       []*nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner:   alice.Address(),
			id:      []byte("any1@example.com"),
			details: username.TokenDetails{[]username.PublicKey{myPubKey}},
		},
		{ // without details
			owner: alice.Address(),
			id:    []byte("any2@example.com"),
		},
		{ // not signed by owner
			owner:           bob.Address(),
			id:              []byte("any3@example.com"),
			details:         username.TokenDetails{[]username.PublicKey{myPubKey}},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // id missing
			owner:           alice.Address(),
			details:         username.TokenDetails{[]username.PublicKey{myPubKey}},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // owner missing
			id:              []byte("any3@example.com"),
			details:         username.TokenDetails{[]username.PublicKey{myPubKey}},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // duplicate algorithm
			owner:           alice.Address(),
			id:              []byte("any4@example.com"),
			details:         username.TokenDetails{[]username.PublicKey{myPubKey, {[]byte("other"), "myAlgorithm"}}},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // key already used
			owner:           alice.Address(),
			id:              []byte("existing@example.com"),
			details:         username.TokenDetails{[]username.PublicKey{myPubKey}},
			expCheckError:   false,
			expDeliverError: true,
		},
		// todo: add approval cases
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			tx := helpers.MockTx(&username.IssueTokenMsg{
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
			require.NoError(t, err)

			res, err := handler.Deliver(nil, db, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err := bucket.Get(db, spec.id)
			assert.NoError(t, err)
			assert.NotNil(t, o)
			u, _ := username.AsUsername(o)
			assert.Equal(t, spec.details.Keys, u.GetPubKeys())
			// todo: verify approvals
		})
	}
}
