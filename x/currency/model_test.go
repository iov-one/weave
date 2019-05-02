package currency

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
)

func TestValidateTokenInfo(t *testing.T) {
	cases := map[string]struct {
		TokenInfo *TokenInfo
		WantErr   *errors.Error
	}{
		"valid model": {
			TokenInfo: &TokenInfo{
				Metadata: &weave.Metadata{Schema: 1},
				Name:     "foobar",
			},
			WantErr: nil,
		},
		"missing metadata": {
			TokenInfo: &TokenInfo{
				Name: "foobar",
			},
			WantErr: errors.ErrMetadata,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.TokenInfo.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation errror: %s", err)
			}
		})
	}

}

func TestTokenInfoBucketQuery(t *testing.T) {
	bucket := NewTokenInfoBucket()

	db := store.MemStore()
	migration.MustInitPkg(db, "currency")

	// Registration of invalid token must fail.
	obj := NewTokenInfo("this is not a valid name", "Invalid Token")
	if err := bucket.Save(db, obj); err == nil {
		t.Fatal("want error")
	}

	doge := NewTokenInfo("DOGE", "Doge Coin")
	if err := bucket.Save(db, doge); err != nil {
		t.Fatalf("cannot register doge: %s", err)
	}
	plop := NewTokenInfo("PLP", "Plop Coin")
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
