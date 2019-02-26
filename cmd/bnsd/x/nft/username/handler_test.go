package username_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/common"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, anybody := helpers.MakeKey()
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	nft.RegisterAction(nft.DefaultActions...)

	db := store.MemStore()
	bucket := username.NewBucket()

	o, _ := bucket.Create(db, bob.Address(), []byte("existing@example.com"), nil, []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "bobsChainAddress"}})
	bucket.Save(db, o)

	handler := username.NewIssueHandler(helpers.Authenticate(alice), nil, bucket)
	// when
	myNewChainAddresses := []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "anyChainAddress"}}
	specs := []struct {
		owner, id       []byte
		details         username.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner:   alice.Address(),
			id:      []byte("any1@example.com"),
			details: username.TokenDetails{Addresses: myNewChainAddresses},
		},
		{ // without details
			owner: alice.Address(),
			id:    []byte("any2@example.com"),
		},
		{ // not signed by owner
			owner:           anybody.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // id missing
			owner:           alice.Address(),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // owner missing
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // duplicate chainID
			owner: alice.Address(),
			id:    []byte("any@example.com"),
			details: username.TokenDetails{
				Addresses: append(myNewChainAddresses,
					username.ChainAddress{BlockchainID: []byte("myNet"), Address: "anyOtherAddress"}),
			},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // name already used
			owner:           alice.Address(),
			id:              []byte("existing@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   false,
			expDeliverError: true,
		},
		{ // address already used
			owner:           alice.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{Addresses: []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "bobsChainAddress"}}},
			expCheckError:   false,
			expDeliverError: true,
		},
		{ // valid approvals
			owner:   alice.Address(),
			id:      []byte("any5@example.com"),
			details: username.TokenDetails{Addresses: myNewChainAddresses},
			approvals: []nft.ActionApprovals{{
				Action:    nft.UpdateDetails,
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid approvals
			owner:           alice.Address(),
			id:              []byte("any6@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
			approvals: []nft.ActionApprovals{{
				Action:    "12",
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{}, Address: nil}},
			}},
		},
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := helpers.MockTx(&username.IssueTokenMsg{
				Owner:     spec.owner,
				ID:        spec.id,
				Details:   spec.details,
				Approvals: spec.approvals,
			})

			// when
			_, err := handler.Check(nil, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			// then
			require.NoError(t, err)

			// and when delivered
			res, err := handler.Deliver(nil, cache, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err := bucket.Get(cache, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			if len(spec.details.Addresses) == 0 {
				assert.Len(t, u.GetChainAddresses(), 0)
			} else {
				assert.Equal(t, spec.details.Addresses, u.GetChainAddresses())
			}
		})
	}
}
func TestIssueUsernameTx(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	db := store.MemStore()
	handler := username.NewIssueHandler(helpers.Authenticate(alice), nil, username.NewBucket())

	// when
	tx := helpers.MockTx(&username.IssueTokenMsg{
		Owner:   alice.Address(),
		ID:      []byte("any@example.com"),
		Details: username.TokenDetails{Addresses: []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "myChainAddress"}}},
	})
	res, err := handler.Deliver(nil, db, tx)
	// then
	require.NoError(t, err)
	assert.Equal(t, []common.KVPair{{Key: []byte("msgType"), Value: []byte("registerUsername")}}, res.Tags)

}

func TestQueryUsernameToken(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()
	aliceAddress := []username.ChainAddress{{BlockchainID: []byte("myChainID"), Address: "aliceChainAddress"}}
	bobAddress := []username.ChainAddress{{BlockchainID: []byte("myChainID"), Address: "bobChainAddress"}}

	db := store.MemStore()
	bucket := username.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, aliceAddress)
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("bob@example.com"), nil, bobAddress)
	require.NoError(t, bucket.Save(db, o2))

	qr := weave.NewQueryRouter()
	username.RegisterQuery(qr)

	specs := []struct {
		path        string
		data        []byte
		expUsername string
	}{
		{ // by username alice
			"/nft/usernames", []byte("alice@example.com"), "alice@example.com"},
		{ // by chain address
			"/nft/usernames/chainaddr", []byte("aliceChainAddress;myChainID"), "alice@example.com"},
		{ // by owner
			"/nft/usernames/owner", alice.Address(), "alice@example.com"},
		{ // by username bob
			"/nft/usernames", []byte("bob@example.com"), "bob@example.com"},
		{ // by chain address
			"/nft/usernames/chainaddr", []byte("bobChainAddress;myChainID"), "bob@example.com"},
		{ // by owner
			"/nft/usernames/owner", bob.Address(), "bob@example.com"},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			// when
			h := qr.Handler(spec.path)
			require.NotNil(t, h)
			mods, err := h.Query(db, "", spec.data)

			// then
			require.NoError(t, err)
			require.Len(t, mods, 1)

			assert.Equal(t, bucket.DBKey([]byte(spec.expUsername)), mods[0].Key)
			got, err := bucket.Parse(nil, mods[0].Value)
			require.NoError(t, err)
			_, err = username.AsUsername(got)
			require.NoError(t, err)
		})
	}
}

//TODO: This needs to be extended with examples where we use approvals for different users
func TestAddChainAddress(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := username.NewBucket()

	myAddress := []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "myChainAddress"}}
	o, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddress)
	bucket.Save(db, o)
	myOtherAddress := []username.ChainAddress{{BlockchainID: []byte("myOtherNet"), Address: "myOtherChainAddress"}}
	o, _ = bucket.Create(db, alice.Address(), []byte("anyOther@example.com"), nil, myOtherAddress)
	bucket.Save(db, o)
	myNextAddress := []username.ChainAddress{{BlockchainID: []byte("myNextNet"), Address: "myNextChainAddress"}}
	o, _ = bucket.Create(db,
		bob.Address(),
		[]byte("withcount@example.com"),
		[]nft.ActionApprovals{{
			Action:    nft.UpdateDetails,
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: 10, UntilBlockHeight: 5}, Address: alice.Address()}},
		}}, myNextAddress)
	bucket.Save(db, o)
	handler := username.NewAddChainAddressHandler(helpers.Authenticate(alice), nil, bucket)

	specs := []struct {
		id              []byte
		newAddress      string
		newChainID      []byte
		expCheckError   bool
		expDeliverError bool
		expCount        int64
		ctx             weave.Context
		expApprovals    nft.Approvals
		expChainAddress []username.ChainAddress
	}{
		{ // happy path
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherAddressID",
			expChainAddress: append(myAddress, username.ChainAddress{BlockchainID: []byte("myOtherNet"), Address: "myOtherAddressID"}),
			ctx:             context.Background(),
		},
		{ // empty address
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			expCheckError:   true,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		{ // empty chainID
			id:              []byte("alice@example.com"),
			newAddress:      "myOtherAddressID",
			expCheckError:   true,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		{ // existing chain
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myNet"),
			newAddress:      "myOtherAddressID",
			expCheckError:   false,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		{ // non unique chain address
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherChainAddress",
			expCheckError:   false,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		{ // unknown id
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myUnknownNet"),
			newAddress:      "myOtherAddressID",
			expCheckError:   false,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		{ // happy path with count decremented
			id:         []byte("withcount@example.com"),
			newChainID: []byte("myOtherNet"),
			newAddress: "myOtherAddressID",
			expApprovals: nft.Approvals{
				nft.UpdateDetails: []nft.Approval{
					{Options: nft.ApprovalOptions{Count: 9, UntilBlockHeight: 5}, Address: alice.Address()}}},
			expChainAddress: append(myNextAddress, username.ChainAddress{BlockchainID: []byte("myOtherNet"), Address: "myOtherAddressID"}),
			ctx:             context.Background(),
		},
		{ // approval timeout
			id:              []byte("withcount@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherAddressID",
			ctx:             weave.WithHeight(context.Background(), 10),
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := helpers.MockTx(&username.AddChainAddressMsg{
				UsernameID:   spec.id,
				BlockchainID: spec.newChainID,
				Address:      spec.newAddress,
			})

			// when
			_, err := handler.Check(spec.ctx, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			// then
			require.NoError(t, err)

			// and when delivered
			res, err := handler.Deliver(spec.ctx, cache, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err = bucket.Get(cache, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			// todo: test sorting
			assert.EqualValues(t, spec.expChainAddress, u.GetChainAddresses())

			if len(spec.expApprovals) > 0 {
				assert.False(t, u.Approvals().List().Intersect(spec.expApprovals).IsEmpty())
			}
		})
	}
}

//TODO: This needs to be extended with examples where we use approvals for different users
func TestRemoveChainAddress(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	ctx := context.Background()

	db := store.MemStore()
	bucket := username.NewBucket()

	myAddresses := []username.ChainAddress{{BlockchainID: []byte("myChainID"), Address: "myChainAddress"}, {BlockchainID: []byte("myOtherNet"), Address: "myOtherChainAddress"}}
	o, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddresses)
	bucket.Save(db, o)

	handler := username.NewRemoveChainAddressHandler(helpers.Authenticate(alice), nil, bucket)

	specs := []struct {
		id              []byte
		newAddress      string
		newChainID      []byte
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			id:         []byte("alice@example.com"),
			newChainID: []byte("myChainID"),
			newAddress: "myChainAddress",
		},
		{ // empty address submitted
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myChainID"),
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // empty chainID
			id:              []byte("alice@example.com"),
			newAddress:      "myChainAddress",
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // unknown name
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myNewChainID"),
			newAddress:      "myChainAddress",
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := helpers.MockTx(&username.RemoveChainAddressMsg{
				UsernameID:   spec.id,
				BlockchainID: spec.newChainID,
				Address:      spec.newAddress,
			})

			// when
			_, err := handler.Check(ctx, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			// then
			require.NoError(t, err)

			// and when delivered
			res, err := handler.Deliver(ctx, db, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err = bucket.Get(db, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			assert.Len(t, u.GetChainAddresses(), 1)
		})
	}

}
