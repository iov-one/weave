package validators

import (
	"encoding/json"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHandler(t *testing.T) {
	var helpers x.TestHelpers

	Convey("Test handler works as intended", t, func() {
		addr := []byte{1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2}
		addr2 := []byte{4, 5, 6, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2}

		perm := weave.NewCondition("sig", "ed25519", addr)
		perm2 := weave.NewCondition("sig", "ed25519", addr2)

		auth := helpers.Authenticate(perm)
		auth2 := helpers.Authenticate(perm2)

		accts := WeaveAccounts{[]weave.Address{perm.Address()}}
		accountsJson, err := json.Marshal(accts)
		So(err, ShouldBeNil)

		kv := store.MemStore()
		bucket := NewBucket()
		init := Initializer{}
		err = init.FromGenesis(weave.Options{optKey: accountsJson}, kv)
		So(err, ShouldBeNil)
		ctrl := NewController(bucket)

		Convey("Check Deliver and Check", func() {
			Convey("With a right address", func() {
				tx := helpers.MockTx(&SetValidators{Validators: []*Validator{{}}})
				handler := NewUpdateHandler(auth, ctrl, authCheckAddress)

				_, err := handler.Deliver(nil, kv, tx)
				So(err, ShouldBeNil)

				_, err = handler.Check(nil, kv, tx)
				So(err, ShouldBeNil)
			})

			Convey("With a wrong address", func() {
				tx := helpers.MockTx(&SetValidators{Validators: []*Validator{{}}})
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
