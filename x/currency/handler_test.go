package currency

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestNewTokenInfoHandler(t *testing.T) {
	var helpers x.TestHelpers

	_, permA := helpers.MakeKey()
	_, permB := helpers.MakeKey()

	cases := map[string]struct {
		signers         []weave.Condition
		issuer          weave.Address
		initState       []orm.Object
		msg             weave.Msg
		wantCheckErr    uint32
		wantDeliverErr  uint32
		query           string
		wantQueryResult orm.Object
	}{
		"updating token info": {
			signers: []weave.Condition{permA, permB},
			issuer:  permA.Address(),
			initState: []orm.Object{
				orm.NewSimpleObj([]byte("DOGE"), &TokenInfo{Name: "Doge Coin", SigFigs: 6}),
			},
			msg:            &NewTokenInfoMsg{Ticker: "DOGE", Name: "Doge Coin", SigFigs: 6},
			wantCheckErr:   CodeInvalidToken,
			wantDeliverErr: CodeInvalidToken,
		},
		"insufficient permission": {
			signers:        []weave.Condition{permB},
			issuer:         permA.Address(),
			msg:            &NewTokenInfoMsg{Ticker: "DOGE", Name: "Doge Coin", SigFigs: 6},
			wantCheckErr:   errors.CodeUnauthorized,
			wantDeliverErr: errors.CodeUnauthorized,
		},
		"query unknown ticker": {
			signers:         []weave.Condition{permA, permB},
			issuer:          permA.Address(),
			msg:             &NewTokenInfoMsg{Ticker: "DOGE", Name: "Doge Coin", SigFigs: 6},
			query:           "UNK",
			wantQueryResult: nil,
		},
		"ok": {
			signers:         []weave.Condition{permA, permB},
			issuer:          permA.Address(),
			msg:             &NewTokenInfoMsg{Ticker: "TKR", Name: "tikr", SigFigs: 6},
			query:           "TKR",
			wantQueryResult: orm.NewSimpleObj([]byte("TKR"), &TokenInfo{Name: "tikr", SigFigs: 6}),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			bucket := NewTokenInfoBucket()
			for _, obj := range tc.initState {
				if err := bucket.Save(db, obj); err != nil {
					t.Fatalf("init state: cannot save: %s", err)
				}
			}

			auth := helpers.Authenticate(tc.signers...)
			h := NewTokenInfoHandler(auth, tc.issuer)
			tx := helpers.MockTx(tc.msg)
			if _, err := h.Check(nil, db, tx); errcode(err) != tc.wantCheckErr {
				t.Fatalf("check error: want %d, got %+v", tc.wantCheckErr, err)
			}
			if _, err := h.Deliver(nil, db, tx); errcode(err) != tc.wantDeliverErr {
				t.Fatalf("deliver error: want %d, got %+v", tc.wantCheckErr, err)
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

// errcode extract an error code from given error or returns 0.
func errcode(err error) uint32 {
	if err == nil {
		return 0
	}
	if e, ok := err.(errors.TMError); ok {
		return e.ABCICode()
	}
	return 0
}
