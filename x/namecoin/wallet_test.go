package namecoin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
)

// BadBucket contains objects that won't satisfy Coinage interface
type BadBucket struct {
	orm.Bucket
}

func (b BadBucket) GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error) {
	// always create....
	return orm.NewSimpleObj(nil, new(Token)), nil
}

// TestValidateWalletBucket makes sure we enforce proper bucket contents
// on init.
func TestValidateWalletBucket(t *testing.T) {
	wb := NewWalletBucket()
	cb := BadBucket{orm.NewBucket("foo", orm.NewSimpleObj(nil, new(Token)))}
	// make sure this doesn't panic
	assert.NotPanics(t, func() { cash.ValidateWalletBucket(wb) })
	assert.Panics(t, func() { cash.ValidateWalletBucket(cb) })

	// make sure save errors on bad object
	db := store.MemStore()
	addr := weave.NewAddress([]byte{17, 93})
	err := wb.Save(db, orm.NewSimpleObj(addr, new(Token)))
	require.Error(t, err)
}

func TestWalletBucket(t *testing.T) {
	bucket := NewWalletBucket()
	addr := weave.NewAddress([]byte{1, 2, 3, 4})

	coin := x.NewCoin(100, 0, "RTC")
	coins := []*x.Coin{&coin}
	alice := &Wallet{Name: "alice", Coins: coins}

	cases := []struct {
		set      []orm.Object
		setError bool
		queries  []weave.Address
		expected []*Wallet
	}{
		// empty
		0: {nil, false, []weave.Address{addr}, []*Wallet{nil}},
		// reject wrong type
		1: {[]orm.Object{NewToken("ERC", "Special", 8)}, true, nil, nil},
		// reject invalid wallets - no address
		2: {[]orm.Object{NewWallet(nil)}, true, nil, nil},
		// allow empty wallet
		3: {[]orm.Object{NewWallet(addr)}, false, []weave.Address{addr}, []*Wallet{&Wallet{}}},
		// invalid name
		4: {
			[]orm.Object{orm.NewSimpleObj(addr,
				&Wallet{Name: "(YB)nu2(*^%", Coins: coins})},
			true, nil, nil},
		// valid
		5: {
			[]orm.Object{orm.NewSimpleObj(addr, alice)},
			false, []weave.Address{addr}, []*Wallet{alice}},
		// // query works fine with one or two tokens
		// 5: {
		// 	[]orm.Object{NewToken("ABC", "Michael", 5)},
		// 	false,
		// 	[]string{"ABC", "LED"},
		// 	[]*Token{&Token{"Michael", 5}, nil},
		// },
		// 6: {
		// 	[]orm.Object{
		// 		NewToken("ABC", "Jackson", 5),
		// 		NewToken("LED", "Zeppelin", 4),
		// 	},
		// 	false,
		// 	[]string{"ABC", "LED"},
		// 	[]*Token{&Token{"Michael", 5}, &Token{"Zeppelin", 4}},
		// },
		// // cannot double-create tokens
		// 7: {
		// 	[]orm.Object{
		// 		NewToken("ABC", "Michael", 5),
		// 		NewToken("ABC", "Jackson", 8),
		// 	},
		// 	true, nil, nil,
		// },
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			err := saveAll(bucket, db, tc.set)
			if tc.setError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			for j, q := range tc.queries {
				obj, err := bucket.Get(db, q)
				require.NoError(t, err)
				if obj != nil {
					assert.EqualValues(t, q, obj.Key())
				}
				assert.EqualValues(t, tc.expected[j], AsWallet(obj), "%x", q)
			}
		})
	}
}
