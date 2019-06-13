package username_test

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/common"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	anybody := weavetest.NewCondition()
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()

	nft.RegisterAction(nft.DefaultActions...)

	db := store.MemStore()
	bucket := username.NewBucket()

	bobsAddress := username.ChainAddress{
		BlockchainID: []byte("myNet"),
		Address:      "bobsChainAddress",
	}
	o, _ := bucket.Create(db, bob.Address(), []byte("existing@example.com"), nil, []username.ChainAddress{bobsAddress})
	bucket.Save(db, o)

	handler := username.NewIssueHandler(
		&weavetest.Auth{Signer: alice}, nil, bucket)
	myNewChainAddresses := []username.ChainAddress{{BlockchainID: []byte("myNet"), Address: "anyChainAddress"}}
	specs := map[string]struct {
		owner, id       []byte
		details         username.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		"happy path": {
			owner:   alice.Address(),
			id:      []byte("any1@example.com"),
			details: username.TokenDetails{Addresses: myNewChainAddresses},
		},
		"without details": {
			owner: alice.Address(),
			id:    []byte("any2@example.com"),
		},
		"not signed by owner": {
			owner:           anybody.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		"id missing": {
			owner:           alice.Address(),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		"owner missing": {
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		"duplicate chainID": {
			owner: alice.Address(),
			id:    []byte("any@example.com"),
			details: username.TokenDetails{
				Addresses: append(myNewChainAddresses,
					username.ChainAddress{BlockchainID: []byte("myNet"), Address: "anyOtherAddress"}),
			},
			expCheckError:   true,
			expDeliverError: true,
		},
		"name already used": {
			owner:           alice.Address(),
			id:              []byte("existing@example.com"),
			details:         username.TokenDetails{Addresses: myNewChainAddresses},
			expCheckError:   false,
			expDeliverError: true,
		},
		"address can have any number of names assigned": {
			owner: alice.Address(),
			id:    []byte("any@example.com"),
			details: username.TokenDetails{
				Addresses: []username.ChainAddress{
					bobsAddress,
				},
			},
			expCheckError:   false,
			expDeliverError: false,
		},
		"valid approvals": {
			owner:   alice.Address(),
			id:      []byte("any5@example.com"),
			details: username.TokenDetails{Addresses: myNewChainAddresses},
			approvals: []nft.ActionApprovals{{
				Action:    nft.UpdateDetails,
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		"invalid approvals": {
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

	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := &weavetest.Tx{
				Msg: &username.IssueTokenMsg{
					Metadata:  &weave.Metadata{Schema: 1},
					Owner:     spec.owner,
					ID:        spec.id,
					Details:   spec.details,
					Approvals: spec.approvals,
				},
			}

			_, err := handler.Check(nil, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			res, err := handler.Deliver(nil, cache, tx)
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

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
	alice := weavetest.NewCondition()

	db := store.MemStore()
	handler := username.NewIssueHandler(
		&weavetest.Auth{Signer: alice}, nil, username.NewBucket())

	tx := &weavetest.Tx{
		Msg: &username.IssueTokenMsg{
			Owner: alice.Address(),
			ID:    []byte("any@example.com"),
			Details: username.TokenDetails{
				Addresses: []username.ChainAddress{
					{
						BlockchainID: []byte("myNet"),
						Address:      "myChainAddress",
					},
				},
			},
		},
	}
	res, err := handler.Deliver(nil, db, tx)
	require.NoError(t, err)
	assert.Equal(t, []common.KVPair{{Key: []byte("msgType"), Value: []byte("registerUsername")}}, res.Tags)

}

func TestQueryUsernameToken(t *testing.T) {
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()
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

	specs := map[string]struct {
		path        string
		data        []byte
		expUsername string
	}{
		"by username alice": {
			path:        "/nft/usernames",
			data:        []byte("alice@example.com"),
			expUsername: "alice@example.com",
		},
		"by owner alice": {
			path:        "/nft/usernames/owner",
			data:        alice.Address(),
			expUsername: "alice@example.com",
		},
		"by username bob": {
			path:        "/nft/usernames",
			data:        []byte("bob@example.com"),
			expUsername: "bob@example.com",
		},
		"by owner bob": {
			path:        "/nft/usernames/owner",
			data:        bob.Address(),
			expUsername: "bob@example.com",
		},
	}
	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			h := qr.Handler(spec.path)
			require.NotNil(t, h)
			mods, err := h.Query(db, "", spec.data)

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
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()

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
	handler := username.NewAddChainAddressHandler(
		&weavetest.Auth{Signer: alice}, nil, bucket)

	specs := map[string]struct {
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
		"happy path": {
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherAddressID",
			expChainAddress: append(myAddress, username.ChainAddress{BlockchainID: []byte("myOtherNet"), Address: "myOtherAddressID"}),
			ctx:             context.Background(),
		},
		"empty address": {
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			expCheckError:   true,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		"empty chainID": {
			id:              []byte("alice@example.com"),
			newAddress:      "myOtherAddressID",
			expCheckError:   true,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		"existing chain": {
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myNet"),
			newAddress:      "myOtherAddressID",
			expCheckError:   false,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		"an address can have more than one alias": {
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherChainAddress",
			expCheckError:   false,
			expDeliverError: false,
			expChainAddress: []username.ChainAddress{
				{BlockchainID: []byte("myNet"), Address: "myChainAddress"},
				{BlockchainID: []byte("myOtherNet"), Address: "myOtherChainAddress"},
			},
			ctx: context.Background(),
		},
		"unknown id": {
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myUnknownNet"),
			newAddress:      "myOtherAddressID",
			expCheckError:   false,
			expDeliverError: true,
			ctx:             context.Background(),
		},
		"happy path with count decremented": {
			id:         []byte("withcount@example.com"),
			newChainID: []byte("myOtherNet"),
			newAddress: "myOtherAddressID",
			expApprovals: nft.Approvals{
				nft.UpdateDetails: []nft.Approval{
					{Options: nft.ApprovalOptions{Count: 9, UntilBlockHeight: 5}, Address: alice.Address()}}},
			expChainAddress: append(myNextAddress, username.ChainAddress{BlockchainID: []byte("myOtherNet"), Address: "myOtherAddressID"}),
			ctx:             context.Background(),
		},
		"approval timeout": {
			id:              []byte("withcount@example.com"),
			newChainID:      []byte("myOtherNet"),
			newAddress:      "myOtherAddressID",
			ctx:             weave.WithHeight(context.Background(), 10),
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := &weavetest.Tx{
				Msg: &username.AddChainAddressMsg{
					UsernameID:   spec.id,
					BlockchainID: spec.newChainID,
					Address:      spec.newAddress,
				},
			}

			_, err := handler.Check(spec.ctx, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			cache.Discard()

			res, err := handler.Deliver(spec.ctx, cache, tx)
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// Ensure entity is persisted.
			o, err = bucket.Get(cache, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			assert.EqualValues(t, spec.expChainAddress, u.GetChainAddresses())

			if len(spec.expApprovals) > 0 {
				assert.False(t, u.Approvals().List().Intersect(spec.expApprovals).IsEmpty())
			}
		})
	}
}

//TODO: This needs to be extended with examples where we use approvals for different users
func TestRemoveChainAddress(t *testing.T) {
	alice := weavetest.NewCondition()
	ctx := context.Background()

	db := store.MemStore()
	bucket := username.NewBucket()

	myAddresses := []username.ChainAddress{{BlockchainID: []byte("myChainID"), Address: "myChainAddress"}, {BlockchainID: []byte("myOtherNet"), Address: "myOtherChainAddress"}}
	o, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddresses)
	bucket.Save(db, o)

	handler := username.NewRemoveChainAddressHandler(
		&weavetest.Auth{Signer: alice}, nil, bucket)

	cases := map[string]struct {
		id              []byte
		newAddress      string
		newChainID      []byte
		expCheckError   bool
		expDeliverError bool
	}{
		"happy path": {
			id:         []byte("alice@example.com"),
			newChainID: []byte("myChainID"),
			newAddress: "myChainAddress",
		},
		"empty address submitted": {
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myChainID"),
			expCheckError:   true,
			expDeliverError: true,
		},
		"empty chainID": {
			id:              []byte("alice@example.com"),
			newAddress:      "myChainAddress",
			expCheckError:   true,
			expDeliverError: true,
		},
		"unknown name": {
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myNewChainID"),
			newAddress:      "myChainAddress",
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for testName, spec := range cases {
		t.Run(testName, func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := &weavetest.Tx{
				Msg: &username.RemoveChainAddressMsg{
					UsernameID:   spec.id,
					BlockchainID: spec.newChainID,
					Address:      spec.newAddress,
				},
			}

			_, err := handler.Check(ctx, cache, tx)
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			res, err := handler.Deliver(ctx, db, tx)
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// Ensure entity is persisted.
			o, err = bucket.Get(db, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			assert.Len(t, u.GetChainAddresses(), 1)
		})
	}

}
