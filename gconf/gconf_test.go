package gconf

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSaveLoad(t *testing.T) {
	cases := map[string]struct {
		Conf        interface{}
		Want        interface{}
		WantSaveErr *errors.Error
		WantLoadErr *errors.Error
	}{
		"string": {
			Conf: &struct{ Value string }{Value: "foobar"},
			Want: &struct{ Value string }{},
		},
		"int64": {
			Conf: &struct{ Value int64 }{Value: 852151421},
			Want: &struct{ Value int64 }{},
		},
		"coin": {
			Conf: &struct{ Value coin.Coin }{Value: coin.NewCoin(51, 924, "IOV")},
			Want: &struct{ Value coin.Coin }{},
		},
		"coin pointer": {
			Conf: &struct{ Value *coin.Coin }{Value: coin.NewCoinp(51, 924, "IOV")},
			Want: &struct{ Value *coin.Coin }{},
		},
		"coin nil pointer": {
			Conf: &struct{ Value *coin.Coin }{Value: nil},
			Want: &struct{ Value *coin.Coin }{},
		},
		"address": {
			Conf: &struct{ Value weave.Address }{Value: weavetest.RandomAddr(t)},
			Want: &struct{ Value weave.Address }{},
		},
		"invalid address cannot be saved": {
			Conf:        &struct{ Value weave.Address }{Value: weave.Address("too short")},
			WantSaveErr: errors.ErrInvalidInput,
		},
		"invalid coin cannot be saved": {
			Conf:        &struct{ Value coin.Coin }{Value: coin.Coin{}},
			WantSaveErr: errors.ErrCurrency,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			if err := Save(db, tc.Conf); !tc.WantSaveErr.Is(err) {
				t.Fatalf("unexpected save error: %s", err)
			}
			if tc.WantSaveErr != nil {
				return
			}

			if err := Load(db, tc.Want); !tc.WantLoadErr.Is(err) {
				t.Fatalf("cannot load configuration: %s", err)
			}
			if tc.WantSaveErr != nil {
				return
			}

			assert.Equal(t, tc.Conf, tc.Want)
		})
	}
}

type MyConfig struct {
	Number int64
	Text   string
	Addr   weave.Address
}
