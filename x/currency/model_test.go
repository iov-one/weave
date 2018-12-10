package currency

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/store"
)

func TestTokenInfoBucketQuery(t *testing.T) {
	bucket := NewTokenInfoBucket()

	db := store.MemStore()

	// Registration of invalid token must fail.
	obj := NewTokenInfo("this is not a valid name", "Invalid Token", 4)
	if err := bucket.Save(db, obj); err == nil {
		t.Fatal("want error")
	}

	doge := NewTokenInfo("DOGE", "Doge Coin", 4)
	if err := bucket.Save(db, doge); err != nil {
		t.Fatalf("cannot register doge: %s", err)
	}
	plop := NewTokenInfo("PLP", "Plop Coin", 7)
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
