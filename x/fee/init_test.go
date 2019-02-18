package fee

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGenesis(t *testing.T) {
	Convey("Test initializer", t, func() {
		genesis := `
		{
			"fees": [
				{"id": "currency/tokeninfo",  "fee": {"whole":50,"fractional":1234567}},
				{"id": "currency/tokeninfo:IOV",  "fee": {"whole":60,"fractional":1234577}}
			]
		}`
		var o weave.Options

		err := json.Unmarshal([]byte(genesis), &o)
		So(err, ShouldBeNil)

		db := store.MemStore()

		var init Initializer
		err = init.FromGenesis(o, db)
		So(err, ShouldBeNil)

		bucket := NewTransactionFeeBucket()

		obj, err := bucket.Get(db, "currency/tokeninfo")
		So(err, ShouldBeNil)
		So(obj, ShouldNotBeNil)

		Convey("Match data in the object", func() {
			fee := obj.Value().(*TransactionFee)

			So(fee.Fee.Fractional, ShouldEqual, 1234567)
			So(fee.Fee.Whole, ShouldEqual, 50)
			So(fee.Fee.Ticker, ShouldEqual, "IOV")
		})
	})
}
