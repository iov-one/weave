package currency

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestNewTokenInfoHandler(t *testing.T) {
	permA := weavetest.NewCondition()
	permB := weavetest.NewCondition()

	cases := map[string]struct {
		signers         []weave.Condition
		issuer          weave.Address
		initState       []orm.Object
		msg             weave.Msg
		wantCheckErr    *errors.Error
		wantDeliverErr  *errors.Error
		query           string
		wantQueryResult orm.Object
	}{
		"updating token info": {
			signers: []weave.Condition{permA, permB},
			issuer:  permA.Address(),
			initState: []orm.Object{
				orm.NewSimpleObj([]byte("DOGE"), &TokenInfo{
					Metadata: &weave.Metadata{Schema: 1},
					Name:     "Doge Coin",
				}),
			},
			msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "DOGE",
				Name:     "Doge Coin",
			},
			wantCheckErr:   errors.ErrDuplicate,
			wantDeliverErr: errors.ErrDuplicate,
		},
		"insufficient permission": {
			signers: []weave.Condition{permB},
			issuer:  permA.Address(),
			msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "DOGE",
				Name:     "Doge Coin",
			},
			wantCheckErr:   errors.ErrUnauthorized,
			wantDeliverErr: errors.ErrUnauthorized,
		},
		"query unknown ticker": {
			signers: []weave.Condition{permA, permB},
			issuer:  permA.Address(),
			msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "DOGE",
				Name:     "Doge Coin",
			},
			query:           "UNK",
			wantQueryResult: nil,
		},
		"ok": {
			signers: []weave.Condition{permA, permB},
			issuer:  permA.Address(),
			msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Ticker:   "TKR",
				Name:     "tikr",
			},
			query: "TKR",
			wantQueryResult: orm.NewSimpleObj([]byte("TKR"), &TokenInfo{
				Metadata: &weave.Metadata{Schema: 1},
				Name:     "tikr",
			}),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "currency")
			bucket := NewTokenInfoBucket()
			for _, obj := range tc.initState {
				if err := bucket.Save(db, obj); err != nil {
					t.Fatalf("init state: cannot save: %s", err)
				}
			}

			auth := &weavetest.Auth{Signers: tc.signers}
			h := newCreateTokenInfoHandler(auth, tc.issuer)
			tx := &weavetest.Tx{Msg: tc.msg}
			_, err := h.Check(nil, db, tx)
			if err != nil {
				if !tc.wantCheckErr.Is(err) {
					t.Fatalf("check error: want %v, got %+v", tc.wantCheckErr, err)
				}
			}
			_, err = h.Deliver(nil, db, tx)
			if err != nil {
				if !tc.wantDeliverErr.Is(err) {
					t.Fatalf("deliver error: want %v, got %+v", tc.wantDeliverErr, err)
				}
			}

			if res, err := bucket.Get(db, tc.query); err != nil {
				t.Fatalf("query failed: %s", err)
			} else if !reflect.DeepEqual(res, tc.wantQueryResult) {
				t.Logf("want: %#v", tc.wantQueryResult)
				t.Logf(" got: %#v", res)
				t.Fatal("unexpected query result")
			}
		})
	}
}
