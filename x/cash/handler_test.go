package cash

import (
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

func TestSend(t *testing.T) {
	foo := coin.NewCoin(100, 0, "FOO")
	some := coin.NewCoin(300, 0, "SOME")

	perm := weave.NewCondition("sig", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sig", "ed25519", []byte{4, 5, 6})

	cases := map[string]struct {
		signers        []weave.Condition
		initState      []orm.Object
		msg            weave.Msg
		wantCheckErr   *errors.Error
		wantDeliverErr *errors.Error
	}{
		"nil message": {
			wantCheckErr:   errors.ErrState,
			wantDeliverErr: errors.ErrState,
		},
		"empty message": {
			msg:            &SendMsg{},
			wantCheckErr:   errors.ErrAmount,
			wantDeliverErr: errors.ErrAmount,
		},
		"unauthorized": {
			msg: &SendMsg{
				Amount:      &foo,
				Source:      perm.Address(),
				Destination: perm2.Address(),
			},
			wantCheckErr:   errors.ErrUnauthorized,
			wantDeliverErr: errors.ErrUnauthorized,
		},
		"source has no account": {
			signers: []weave.Condition{perm},
			msg: &SendMsg{
				Amount:      &foo,
				Source:      perm.Address(),
				Destination: perm2.Address(),
			},
			wantDeliverErr: errors.ErrEmpty,
		},
		"source too poor": {
			signers: []weave.Condition{perm},
			initState: []orm.Object{
				must(WalletWith(perm.Address(), &some)),
			},
			msg: &SendMsg{
				Amount:      &foo,
				Source:      perm.Address(),
				Destination: perm2.Address(),
			},
			wantDeliverErr: errors.ErrAmount,
		},
		"source got cash": {
			signers: []weave.Condition{perm},
			initState: []orm.Object{
				must(WalletWith(perm.Address(), &foo)),
			},
			msg: &SendMsg{
				Amount:      &foo,
				Source:      perm.Address(),
				Destination: perm2.Address(),
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			controller := NewController(NewBucket())
			h := NewSendHandler(auth, controller)

			kv := store.MemStore()
			migration.MustInitPkg(kv, "cash")
			bucket := NewBucket()
			for _, wallet := range tc.initState {
				if err := bucket.Save(kv, wallet); err != nil {
					t.Fatalf("cannot save %q wallet: %s", wallet.Key(), err)
				}
			}

			tx := &weavetest.Tx{Msg: tc.msg}

			if _, err := h.Check(nil, kv, tx); !tc.wantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %+v", err)
			}
			if _, err := h.Deliver(nil, kv, tx); !tc.wantDeliverErr.Is(err) {
				t.Fatalf("unexpected deliver error: %+v", err)
			}
		})
	}
}
