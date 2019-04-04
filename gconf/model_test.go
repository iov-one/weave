package gconf

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

func TestBucketPrimitiveTypesGetSet(t *testing.T) {
	cases := map[string]struct {
		val     interface{}
		wantErr *errors.Error
	}{
		"int64": {
			val: int64(942),
		},
		"string": {
			val: "foobar",
		},
		"bytes slice": {
			val: []byte("foobar"),
		},
		"coin": {
			val: coin.NewCoin(1, 2, "IOV"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			b := NewConfBucket()
			propName := []byte(t.Name())

			obj, err := NewConf(propName, tc.val)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected new conf error: %s", err)
			}
			if tc.wantErr != nil {
				return
			}

			if err := b.Save(db, obj); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			var loadVal interface{}
			err = b.Load(db, propName, &loadVal)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected load error: %s", err)
			}

			if !reflect.DeepEqual(tc.val, loadVal) {
				t.Fatalf("unexpected load result: %#v", loadVal)
			}
		})
	}

}

func TestSetLoadAddress(t *testing.T) {
	db := store.MemStore()
	b := NewConfBucket()
	propName := []byte(t.Name())

	addr := hexDecode(t, "a656a66d09a9c810019f7f96c91f423ccf81326f")

	obj, err := NewConf(propName, addr)
	if err != nil {
		t.Fatalf("cannot create an address conf object: %s", err)
	}
	if err := b.Save(db, obj); err != nil {
		t.Fatalf("cannot store the conf object in database: %s", err)
	}

	var a weave.Address
	if err := b.Load(db, propName, &a); err != nil {
		t.Fatalf("cannot load address: %s", err)
	}
	if !a.Equals(addr) {
		t.Fatalf("unexpected address: %q", a)
	}
}

func hexDecode(t testing.TB, s string) weave.Address {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("cannot hex decode: %s", err)
	}
	return weave.Address(raw)
}

func TestSetLoadCoinValue(t *testing.T) {
	db := store.MemStore()
	b := NewConfBucket()
	propName := []byte(t.Name())

	obj, err := NewConf(propName, coin.NewCoin(1, 2, "IOV"))
	if err != nil {
		t.Fatalf("cannot create a Coin conf object: %s", err)
	}
	if err := b.Save(db, obj); err != nil {
		t.Fatalf("cannot store the conf object in database: %s", err)
	}

	// It does not matter how the coin instance was created (using value or
	// address). It can be unloaded to both pointer and address.

	var c coin.Coin
	if err := b.Load(db, propName, &c); err != nil {
		t.Fatalf("cannot load coin: %s", err)
	}
	if !c.Equals(coin.NewCoin(1, 2, "IOV")) {
		t.Fatalf("unexpected coin: %+v", c)
	}

	var cp *coin.Coin
	if err := b.Load(db, propName, &cp); err != nil {
		t.Fatalf("cannot load coin: %s", err)
	}
	if !cp.Equals(coin.NewCoin(1, 2, "IOV")) {
		t.Fatalf("unexpected coin: %+v", cp)
	}
}

func TestSetLoadCoinPointer(t *testing.T) {
	db := store.MemStore()
	b := NewConfBucket()
	propName := []byte(t.Name())

	obj, err := NewConf(propName, coin.NewCoinp(1, 2, "IOV"))
	if err != nil {
		t.Fatalf("cannot create a Coin conf object: %s", err)
	}
	if err := b.Save(db, obj); err != nil {
		t.Fatalf("cannot store the conf object in database: %s", err)
	}

	// It does not matter how the coin instance was created (using value or
	// address). It can be unloaded to both pointer and address.

	var c coin.Coin
	if err := b.Load(db, propName, &c); err != nil {
		t.Fatalf("cannot load coin: %s", err)
	}
	if !c.Equals(coin.NewCoin(1, 2, "IOV")) {
		t.Fatalf("unexpected coin: %+v", c)
	}

	var cp *coin.Coin
	if err := b.Load(db, propName, &cp); err != nil {
		t.Fatalf("cannot load coin: %s", err)
	}
	if !cp.Equals(coin.NewCoin(1, 2, "IOV")) {
		t.Fatalf("unexpected coin: %+v", cp)
	}
}
