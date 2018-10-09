package username_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, anybody := helpers.MakeKey()
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := username.NewBucket()
	blockchains := blockchain.NewBucket()
	b, _ := blockchains.Create(db, anybody.Address(), []byte("myNet"), nil, nil)
	blockchains.Save(db, b)

	o, _ := bucket.Create(db, bob.Address(), []byte("existing@example.com"), nil, []username.ChainAddress{{ChainID: []byte("myNet"), Address: []byte("bobsChainAddress")}})
	bucket.Save(db, o)

	handler := username.NewIssueHandler(helpers.Authenticate(alice), nil, bucket, blockchains)
	// when
	myNewChainAddresses := []username.ChainAddress{{ChainID: []byte("myNet"), Address: []byte("anyChainAddress")}}
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
			details: username.TokenDetails{myNewChainAddresses},
		},
		{ // without details
			owner: alice.Address(),
			id:    []byte("any2@example.com"),
		},
		{ // not signed by owner
			owner:           anybody.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // id missing
			owner:           alice.Address(),
			details:         username.TokenDetails{myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // owner missing
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{myNewChainAddresses},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // duplicate chainID
			owner:           alice.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{append(myNewChainAddresses, username.ChainAddress{[]byte("myNet"), []byte("anyOtherAddress")})},
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // name already used
			owner:           alice.Address(),
			id:              []byte("existing@example.com"),
			details:         username.TokenDetails{myNewChainAddresses},
			expCheckError:   false,
			expDeliverError: true,
		},
		{ // address already used
			owner:           alice.Address(),
			id:              []byte("any@example.com"),
			details:         username.TokenDetails{[]username.ChainAddress{{[]byte("myNet"), []byte("bobsChainAddress")}}},
			expCheckError:   false,
			expDeliverError: true,
		},
		{ // valid approvals
			owner:   alice.Address(),
			id:      []byte("any5@example.com"),
			details: username.TokenDetails{myNewChainAddresses},
			approvals: []nft.ActionApprovals{{
				Action:    nft.Action_ActionUpdateDetails.String(),
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid approvals
			owner:           alice.Address(),
			id:              []byte("any6@example.com"),
			details:         username.TokenDetails{myNewChainAddresses},
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
				Id:        spec.id,
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

func TestQueryTokenByName(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()
	myAddresses := []username.ChainAddress{{ChainID: []byte("myChainID"), Address: []byte("myAddressID0")}}

	db := store.MemStore()
	bucket := username.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddresses)
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("bob@example.com"), nil, myAddresses)
	bucket.Save(db, o2)

	qr := weave.NewQueryRouter()
	username.RegisterQuery(qr)
	// when
	h := qr.Handler("/nft/usernames")
	require.NotNil(t, h)
	mods, err := h.Query(db, "", []byte("alice@example.com"))
	// then
	require.NoError(t, err)
	require.Len(t, mods, 1)

	assert.Equal(t, bucket.DBKey([]byte("alice@example.com")), mods[0].Key)
	got, err := bucket.Parse(nil, mods[0].Value)
	require.NoError(t, err)
	x, err := username.AsUsername(got)
	require.NoError(t, err)
	assert.Equal(t, myAddresses, x.GetChainAddresses())
}

//TODO: This needs to be extended with examples where we use approvals for different users
func TestAddChainAddress(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	db := store.MemStore()
	bucket := username.NewBucket()
	blockchains := blockchain.NewBucket()
	for _, blockchainID := range []string{"myNet", "myOtherNet"} {
		b, _ := blockchains.Create(db, alice.Address(), []byte(blockchainID), nil, nil)
		blockchains.Save(db, b)
	}

	myAddress := []username.ChainAddress{{ChainID: []byte("myNet"), Address: []byte("myChainAddress")}}
	o, err := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddress)
	require.NoError(t, err)
	bucket.Save(db, o)

	handler := username.NewAddChainAddressHandler(helpers.Authenticate(alice), nil, bucket, blockchains)

	specs := []struct {
		id              []byte
		newAddress      []byte
		newChainID      []byte
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			id:         []byte("alice@example.com"),
			newChainID: []byte("myOtherNet"),
			newAddress: []byte("myOtherAddressID"),
		},
		{ // empty address
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myOtherNet"),
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // empty chainID
			id:              []byte("alice@example.com"),
			newAddress:      []byte("myOtherAddressID"),
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // existing chain
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myNet"),
			newAddress:      []byte("myOtherAddressID"),
			expCheckError:   false,
			expDeliverError: true,
		},
		{ // unknown id
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myUnknownNet"),
			newAddress:      []byte("myOtherAddressID"),
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := helpers.MockTx(&username.AddChainAddressMsg{
				Id:      spec.id,
				ChainID: spec.newChainID,
				Address: spec.newAddress,
			})

			// when
			_, err = handler.Check(nil, cache, tx)
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
			o, err = bucket.Get(cache, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			// todo: test sorting
			assert.Equal(t, append(myAddress, username.ChainAddress{spec.newChainID, spec.newAddress}), u.GetChainAddresses())
		})
	}

}

//TODO: This needs to be extended with examples where we use approvals for different users
func TestRemoveChainAddress(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	db := store.MemStore()
	bucket := username.NewBucket()

	myAddresses := []username.ChainAddress{{ChainID: []byte("myChainID"), Address: []byte("myChainAddress")}, {ChainID: []byte("myOtherNet"), Address: []byte("myOtherChainAddress")}}
	o, _ := bucket.Create(db, alice.Address(), []byte("alice@example.com"), nil, myAddresses)
	bucket.Save(db, o)

	handler := username.NewRemoveChainAddressHandler(helpers.Authenticate(alice), nil, bucket)

	specs := []struct {
		id              []byte
		newAddress      []byte
		newChainID      []byte
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			id:         []byte("alice@example.com"),
			newChainID: []byte("myChainID"),
			newAddress: []byte("myChainAddress"),
		},
		{ // empty address submitted
			id:              []byte("alice@example.com"),
			newChainID:      []byte("myChainID"),
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // empty chainID
			id:              []byte("alice@example.com"),
			newAddress:      []byte("myChainAddress"),
			expCheckError:   true,
			expDeliverError: true,
		},
		{ // unknown name
			id:              []byte("unknown@example.com"),
			newChainID:      []byte("myNewChainID"),
			newAddress:      []byte("myChainAddress"),
			expCheckError:   false,
			expDeliverError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			cache := db.CacheWrap()
			defer cache.Discard()

			tx := helpers.MockTx(&username.RemoveChainAddressMsg{
				Id:      spec.id,
				ChainID: spec.newChainID,
				Address: spec.newAddress,
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
			o, err = bucket.Get(db, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := username.AsUsername(o)
			assert.Len(t, u.GetChainAddresses(), 1)
		})
	}

}
