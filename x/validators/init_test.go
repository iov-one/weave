package validators

import (
	"github.com/tendermint/tendermint/abci/types"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
)

func TestInitState(t *testing.T) {
	//alice := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}
	//bert := []byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 10, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}
	specs := map[string]struct {
		State       weave.Options
		Params      weave.GenesisParams
		Exp         *WeaveAccounts
		ValidParams bool
		ExpError    *errors.Error
	}{
		//"Init with addresses": {
		//	State: weave.Options{optKey: []byte(`{"addresses":["0102030405060708090021222324252627282930", "0B0C0D0E0F101112130A21222324252627282930"]}`)},
		//	Exp:   &WeaveAccounts{[]weave.Address{alice, bert}},
		//},
		//"Init works with no appState data": {
		//	State: weave.Options{},
		//},
		//"Init works with no relevant appState data": {
		//	State: weave.Options{"foo": []byte(`"bar"`)},
		//},
		//"Init fails with bad address": {
		//	State:    weave.Options{optKey: []byte(`{"addresses":["00"]}`)},
		//	ExpError: errors.ErrInput,
		//},
		"Init works with correct params": {
			State: weave.Options{},
			Params: weave.GenesisParams{Validators: []types.ValidatorUpdate{
				{Power: 1, PubKey: types.PubKey{Type: "ed25519", Data: make([]byte, 32)}},
			}},
			ValidParams: true,
		},
		//"Init does not work with invalid params": {
		//	State: weave.Options{},
		//	Params: weave.GenesisParams{Validators: []types.ValidatorUpdate{
		//		{Power: 1, PubKey: types.PubKey{Type: "ed25519", Data: make([]byte, 31)}},
		//	}},
		//	ExpError: errors.ErrType,
		//},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			kv := store.MemStore()
			migration.MustInitPkg(kv, "validators")
			bucket := NewAccountBucket()
			// when
			err := Initializer{}.FromGenesis(spec.State, spec.Params, kv)
			if !spec.ExpError.Is(err) {
				t.Fatalf("check expected: %v  but got %+v", spec.ExpError, err)
			}

			if spec.ValidParams {
				res, err := kv.Get([]byte(storeKey))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				vu := ValidatorUpdates{}
				err = vu.Unmarshal(res)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				exp := ValidatorUpdatesFromABCI(spec.Params.Validators)

				if !reflect.DeepEqual(ValidatorUpdatesFromABCI(spec.Params.Validators), vu) {
					t.Errorf("expected %v but got %v", exp, vu)
				}
			}

			if spec.Exp == nil {
				return
			}
			accounts, err := bucket.GetAccounts(kv)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := AsAccounts(*spec.Exp), accounts; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}
