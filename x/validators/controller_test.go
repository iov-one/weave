package validators

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/cash"
	. "github.com/smartystreets/goconvey/convey"
	abci "github.com/tendermint/abci/types"
)

func TestController(t *testing.T) {
	Convey("Test controller works as intended", t, func() {
		addr := []byte("12345678901234567890")
		addr2 := []byte("123456")
		accts := WeaveAccounts{[]weave.Address{addr}}

		checkAddress := func(address weave.Address) bool {
			return address.Equals(addr)
		}

		checkAddress2 := func(address weave.Address) bool {
			return address.Equals(addr2)
		}

		accountsJson, err := json.Marshal(accts)
		So(err, ShouldBeNil)

		diff := []abci.Validator{{}}
		emptyDiff := make([]abci.Validator, 0)

		kv := store.MemStore()
		bucket := NewBucket()
		ctrl := BaseController{bucket: bucket}

		Convey("When init is okay", func() {
			init := Initializer{}
			err := init.FromGenesis(weave.Options{optKey: accountsJson}, kv)
			So(err, ShouldBeNil)

			Convey("Everything is in order", func() {
				d, err := ctrl.CanUpdateValidators(kv, checkAddress, diff)
				So(err, ShouldBeNil)
				So(d, ShouldResemble, diff)
			})

			Convey("Accounts type is nil", func() {
				bucket.Delete(kv, []byte(Key))
				//bucket.Save(kv, orm.NewSimpleObj([]byte(Key), set))
				_, err = ctrl.CanUpdateValidators(kv, checkAddress, diff)
				So(err.Error(), ShouldResemble, ErrWrongType(nil).Error())
			})

			Convey("No permission", func() {
				_, err = ctrl.CanUpdateValidators(kv, checkAddress2, diff)
				So(err.Error(), ShouldResemble, errors.ErrUnauthorized().Error())
			})

			Convey("Empty diff", func() {
				_, err := ctrl.CanUpdateValidators(kv, checkAddress, emptyDiff)
				So(err.Error(), ShouldResemble, ErrEmptyDiff().Error())
			})

			Convey("Accounts type is wrong", func() {
				set := new(cash.Set)
				So(err, ShouldBeNil)
				bucket.Delete(kv, []byte(Key))
				kv.Set([]byte(Key), []byte(set.String()))
				_, err = ctrl.CanUpdateValidators(kv, checkAddress, diff)
				So(err.Error(), ShouldResemble, ErrWrongType(set).Error())
			})
		})

		Convey("When init didn't happen", func() {
			Convey("Error on GetAccounts", func() {
				_, err = ctrl.CanUpdateValidators(kv, checkAddress, diff)
				So(err.Error(), ShouldResemble, ErrWrongType(nil).Error())
			})
		})
	})

}
