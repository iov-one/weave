package validators

import (
	"encoding/json"
	"testing"

	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHandler(t *testing.T) {
	var helpers x.TestHelpers

	Convey("Test handler works as intended", t, func() {
		addr := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)
		addr2 := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)

		perm := weave.NewCondition("sig", "ed25519", addr[:])
		perm2 := weave.NewCondition("sig", "ed25519", addr2[:])

		auth := helpers.Authenticate(perm)
		auth2 := helpers.Authenticate(perm2)

		accts := WeaveAccounts{[]weave.Address{perm.Address()}}
		accountsJson, err := json.Marshal(accts)
		So(err, ShouldBeNil)

		kv := store.MemStore()
		init := Initializer{}
		err = init.FromGenesis(weave.Options{optKey: accountsJson}, kv)
		So(err, ShouldBeNil)
		ctrl := NewController()
		anUpdate := []*ValidatorUpdate{
			{Pubkey: Pubkey{Data: addr[:], Type: "ed25519"}, Power: 10},
		}

		Convey("Check Deliver and Check", func() {
			Convey("With a right address", func() {
				tx := helpers.MockTx(&SetValidatorsMsg{ValidatorUpdates: anUpdate})
				handler := NewUpdateHandler(auth, ctrl, authCheckAddress)

				res, err := handler.Deliver(nil, kv, tx)
				So(err, ShouldBeNil)
				So(len(res.Diff), ShouldEqual, 1)

				_, err = handler.Check(nil, kv, tx)
				So(err, ShouldBeNil)
			})

			Convey("With a wrong address", func() {
				tx := helpers.MockTx(&SetValidatorsMsg{ValidatorUpdates: anUpdate})
				handler := NewUpdateHandler(auth2, ctrl, authCheckAddress)

				_, err := handler.Deliver(nil, kv, tx)
				So(err.Error(), ShouldResemble, errors.ErrUnauthorized().Error())

				_, err = handler.Check(nil, kv, tx)
				So(err.Error(), ShouldResemble, errors.ErrUnauthorized().Error())
			})

			Convey("With an invalid message", func() {
				msg := &cash.SendMsg{}
				tx := helpers.MockTx(msg)
				handler := NewUpdateHandler(auth2, ctrl, authCheckAddress)

				_, err := handler.Deliver(nil, kv, tx)
				So(err.Error(), ShouldResemble, errors.ErrUnknownTxType(msg).Error())

				_, err = handler.Check(nil, kv, tx)
				So(err.Error(), ShouldResemble, errors.ErrUnknownTxType(msg).Error())
			})
		})
	})

}
