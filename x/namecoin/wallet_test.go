package namecoin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/cash"
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
	addr2 := weave.NewAddress([]byte{7, 8, 9, 0})

	c := coin.NewCoin(100, 0, "RTC")
	cs := []*coin.Coin{&c}
	c2 := coin.NewCoin(532, 235, "LRN")
	cs2 := []*coin.Coin{&c2, &c}
	alice := &Wallet{Name: "alice", Coins: cs}
	alice2 := &Wallet{Name: "alice", Coins: cs2}
	bob := &Wallet{Name: "bobby", Coins: cs2}

	cases := []struct {
		set           []orm.Object
		setError      bool
		queries       []weave.Address
		expected      []*Wallet
		queryNames    []string
		expectedNames []*Wallet
	}{
		// empty
		0: {nil, false,
			[]weave.Address{addr}, []*Wallet{nil},
			[]string{"alice"}, []*Wallet{nil},
		},
		// reject wrong type
		1: {[]orm.Object{NewToken("ERC", "Special", 8)}, true,
			nil, nil,
			nil, nil},
		// reject invalid wallets - no address
		2: {[]orm.Object{NewWallet(nil)}, true,
			nil, nil,
			nil, nil},
		// allow empty wallet
		3: {[]orm.Object{NewWallet(addr)}, false,
			[]weave.Address{addr}, []*Wallet{&Wallet{}},
			[]string{"alice"}, []*Wallet{nil},
		},
		// invalid name
		4: {
			[]orm.Object{orm.NewSimpleObj(addr,
				&Wallet{Name: "yo", Coins: cs})},
			true,
			nil, nil,
			nil, nil,
		},
		// valid
		5: {
			[]orm.Object{orm.NewSimpleObj(addr, alice)},
			false,
			[]weave.Address{addr, addr2}, []*Wallet{alice, nil},
			[]string{"alice", "bob"}, []*Wallet{alice, nil},
		},
		// multiple entries
		6: {
			[]orm.Object{
				orm.NewSimpleObj(addr, alice),
				orm.NewSimpleObj(addr2, bob)},
			false,
			[]weave.Address{addr, addr2}, []*Wallet{alice, bob},
			[]string{"alice", "bobby"}, []*Wallet{alice, bob},
		},
		// update one entry with new coins
		7: {
			[]orm.Object{
				orm.NewSimpleObj(addr, alice),
				orm.NewSimpleObj(addr, alice2)},
			false,
			[]weave.Address{addr, addr2}, []*Wallet{alice2, nil},
			[]string{"alice"}, []*Wallet{alice2},
		},
		// same name on two wallets fails
		8: {
			[]orm.Object{
				orm.NewSimpleObj(addr, alice),
				orm.NewSimpleObj(addr2, alice2)},
			true,
			nil, nil,
			nil, nil,
		},
		// TODO: not enforced in bucket, but in handler (SetName)
		// is that enough or should be make this test pass??
		// // update one entry with new name fails
		// 9: {
		// 	[]orm.Object{
		// 		orm.NewSimpleObj(addr, alice),
		// 		orm.NewSimpleObj(addr, bob)},
		// 	true,
		// 	[]weave.Address{addr, addr2},
		// 	[]*Wallet{nil, nil}},
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

			for j, q := range tc.queryNames {
				obj, err := bucket.GetByName(db, q)
				require.NoError(t, err)
				if obj != nil {
					assert.EqualValues(t, q, AsNamed(obj).GetName())
				}
				assert.EqualValues(t, tc.expectedNames[j], AsWallet(obj), q)
			}

		})
	}
}
