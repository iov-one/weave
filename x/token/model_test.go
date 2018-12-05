package token

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
)

func TestTokenInfoBucketQuery(t *testing.T) {
	bucket := NewTokenInfoBucket()

	db := store.MemStore()

	// Registration of invalid token must fail.
	obj := orm.NewSimpleObj([]byte("this is not a valid name"),
		&TokenInfo{Name: "Invalid Token", SigFigs: 4})
	if err := bucket.Save(db, obj); err == nil {
		t.Fatal("want error")
	}

	doge := orm.NewSimpleObj([]byte("DOGE"),
		&TokenInfo{Name: "Doge Coin", SigFigs: 4})
	if err := bucket.Save(db, doge); err != nil {
		t.Fatalf("cannot register doge: %s", err)
	}
	plop := orm.NewSimpleObj([]byte("PLP"),
		&TokenInfo{Name: "Plop Coin", SigFigs: 7})
	if err := bucket.Save(db, plop); err != nil {
		t.Fatalf("cannot register plop: %s", err)
	}

	// Query for an unknown currency must fail.
	if res, err := bucket.Get(db, "XYZ"); err != nil {
		t.Fatal("want fail")
	} else if res != nil {
		t.Fatalf("want nil, got %#v", res)
	}

	if got, err := bucket.Get(db, "DOGE"); err != nil {
		t.Fatalf("cannot query: %s", err)
	} else if want := doge; !reflect.DeepEqual(want, got) {
		t.Logf("want: %#v", want)
		t.Logf(" got: %#v", got)
		t.Fatal("unexpected query result")
	}

	if got, err := bucket.Get(db, "PLP"); err != nil {
		t.Fatalf("cannot query: %s", err)
	} else if want := plop; !reflect.DeepEqual(want, got) {
		t.Logf("want: %#v", want)
		t.Logf(" got: %#v", got)
		t.Fatal("unexpected query result")
	}
}
