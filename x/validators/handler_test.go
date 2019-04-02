package validators

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestHandler(t *testing.T) {
	Convey("Test handler works as intended", t, func() {
		addr1 := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)
		addr2 := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)

		perm1 := weave.NewCondition("sig", "ed25519", addr1[:])
		perm2 := weave.NewCondition("sig", "ed25519", addr2[:])

		auth := &weavetest.Auth{Signer: perm1}
		auth2 := &weavetest.Auth{Signer: perm2}

		accts := WeaveAccounts{[]weave.Address{perm1.Address()}}
		accountsJson, err := json.Marshal(accts)
		So(err, ShouldBeNil)

		kv := store.MemStore()
		init := Initializer{}
		err = init.FromGenesis(weave.Options{optKey: accountsJson}, kv)
		So(err, ShouldBeNil)
		ctrl := NewController()
		anUpdate := []*ValidatorUpdate{
			{Pubkey: Pubkey{Data: addr1[:], Type: "ed25519"}, Power: 10},
		}

		Convey("Check Deliver and Check", func() {
			Convey("With a right address", func() {
				tx := &weavetest.Tx{Msg: &SetValidatorsMsg{ValidatorUpdates: anUpdate}}
				handler := NewUpdateHandler(auth, ctrl, authCheckAddress)

				res, err := handler.Deliver(nil, kv, tx)
				So(err, ShouldBeNil)
				So(len(res.Diff), ShouldEqual, 1)

				_, err = handler.Check(nil, kv, tx)
				So(err, ShouldBeNil)
			})

			Convey("With a wrong address", func() {
				tx := &weavetest.Tx{Msg: &SetValidatorsMsg{ValidatorUpdates: anUpdate}}
				handler := NewUpdateHandler(auth2, ctrl, authCheckAddress)

				_, err := handler.Deliver(nil, kv, tx)
				So(errors.ErrUnauthorized.Is(err), ShouldBeTrue)

				_, err = handler.Check(nil, kv, tx)
				So(errors.ErrUnauthorized.Is(err), ShouldBeTrue)
			})

			//Convey("With an invalid message", func() {
			//	msg := &cash.SendMsg{}
			//	tx := &weavetest.Tx{Msg: msg}
			//	handler := NewUpdateHandler(auth2, ctrl, authCheckAddress)

			//	_, err := handler.Deliver(nil, kv, tx)
			//	So(err.Error(), ShouldResemble, errors.WithType(errors.ErrInvalidAmount, msg).Error())

			//	_, err = handler.Check(nil, kv, tx)
			//	So(err.Error(), ShouldResemble, errors.WithType(errors.ErrInvalidAmount, msg).Error())
			//})
		})
	})

}
