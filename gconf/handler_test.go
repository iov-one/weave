package gconf

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUpdateConfigurationHandler(t *testing.T) {
	cond := weavetest.NewCondition()

	cases := map[string]struct {
		// If Init is provided, initialize the database before running
		// handler code. This should represent the configuration's
		// initial state. Use nil to not provide initial state.
		Init ValidMarshaler

		Msg            weave.Msg
		MsgConditions  []weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error

		// When not nil database state will be tested to contain the
		// exact version of the configuration.
		WantConfig *myconfig
	}{
		"success": {
			Init: &myconfig{
				Owner: cond.Address(),
				Num:   5125,
				Str:   "foobar",
				Cn:    coin.NewCoin(10, 409, "IOV"),
			},
			Msg: &myconfigMsg{
				Patch: &myconfig{
					Owner: cond.Address(),
					Num:   333,
					Str:   "boing!",
					Cn:    coin.NewCoin(4, 4, "XYZ"),
				},
			},
			MsgConditions: []weave.Condition{cond},
			WantConfig: &myconfig{
				Owner: cond.Address(),
				Num:   333,
				Str:   "boing!",
				Cn:    coin.NewCoin(4, 4, "XYZ"),
			},
		},
		"message must be signed by the configuration owner": {
			Init: &myconfig{
				Owner: cond.Address(),
				Num:   5125,
				Str:   "foobar",
				Cn:    coin.NewCoin(10, 409, "IOV"),
			},
			MsgConditions: []weave.Condition{
				// A random condition - for sure not the same as the Owner.
				weavetest.NewCondition(),
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"zero values are not updating the configuration": {
			Init: &myconfig{
				Owner: cond.Address(),
				Num:   5125,
				Str:   "foobar",
				Cn:    coin.NewCoin(10, 409, "IOV"),
			},
			Msg: &myconfigMsg{
				Patch: &myconfig{
					Owner: cond.Address(),
					Num:   0,
					Str:   "",
					Cn:    coin.NewCoin(0, 4, "IOV"),
				},
			},
			MsgConditions: []weave.Condition{cond},
			WantConfig: &myconfig{
				Owner: cond.Address(),
				Num:   5125,
				Str:   "foobar",
				Cn:    coin.NewCoin(0, 4, "IOV"),
			},
		},
		"invalid configuration is not accepted": {
			Init: &myconfig{
				Owner: cond.Address(),
				Num:   5125,
				Str:   "foobar",
				Cn:    coin.NewCoin(10, 409, "IOV"),
			},
			Msg: &myconfigMsg{
				Patch: &myconfig{
					Owner: cond.Address(),
					Num:   123,
					Str:   "foo",
					Cn:    coin.NewCoin(4, 0, ""), // Missing Ticker.
				},
			},
			MsgConditions:  []weave.Condition{cond},
			WantCheckErr:   errors.ErrCurrency,
			WantDeliverErr: errors.ErrCurrency,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			if tc.Init != nil {
				if err := Save(db, "mypkg", tc.Init); err != nil {
					t.Fatalf("cannot save initial configuration: %s", err)
				}
			}

			var c myconfig
			auth := &weavetest.CtxAuth{Key: "auth"}
			handler := NewUpdateConfigurationHandler("mypkg", &c, auth)

			ctx := weave.WithHeight(context.Background(), 999)
			ctx = weave.WithChainID(ctx, "mychain-123")
			ctx = auth.SetConditions(ctx, tc.MsgConditions...)

			tx := &weavetest.Tx{Msg: tc.Msg}

			cache := db.CacheWrap()
			if _, err := handler.Check(ctx, cache, tx); !tc.WantCheckErr.Is(err) {
				t.Fatal(err)
			}
			cache.Discard()

			if _, err := handler.Deliver(ctx, db, tx); !tc.WantDeliverErr.Is(err) {
				t.Fatal(err)
			}

			if tc.WantConfig != nil {
				var got myconfig
				if err := Load(db, "mypkg", &got); err != nil {
					t.Fatalf("cannot load configuration from the database: %s", err)
				}
				assert.Equal(t, tc.WantConfig, &got)
			}
		})
	}
}

type myconfig struct {
	Owner weave.Address
	Num   int64
	Str   string
	Cn    coin.Coin
}

func (c *myconfig) GetOwner() weave.Address    { return c.Owner }
func (c *myconfig) Marshal() ([]byte, error)   { return json.Marshal(c) }
func (c *myconfig) Unmarshal(raw []byte) error { return json.Unmarshal(raw, &c) }

func (c *myconfig) Validate() error {
	if err := c.Owner.Validate(); err != nil {
		return errors.Wrap(err, "address")
	}
	if err := c.Cn.Validate(); err != nil {
		return errors.Wrap(err, "coin")
	}
	return nil
}

type myconfigMsg struct {
	Patch *myconfig
}

var _ weave.Msg = (*myconfigMsg)(nil)

func (msg *myconfigMsg) Marshal() ([]byte, error)   { return json.Marshal(msg) }
func (msg *myconfigMsg) Unmarshal(raw []byte) error { return json.Unmarshal(raw, &msg) }
func (msg *myconfigMsg) Path() string               { return "myconfig" }
func (msg *myconfigMsg) Validate() error            { return msg.Patch.Validate() }
