package validators

import (
	"encoding/json"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInitState(t *testing.T) {
	Convey("Test init state", t, func() {
		// test data
		addr := []byte("12345678901234567890")
		addr2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}
		accts := WeaveAccounts{[]weave.Address{addr}}

		accountsJson, err := json.Marshal(accts)
		So(err, ShouldBeNil)

		accountsJson2 := []byte(`{"addresses":["0102030405060708090021222324252627282930"]}`)
		accts2 := WeaveAccounts{[]weave.Address{addr2}}

		init := Initializer{}

		kv := store.MemStore()
		bucket := NewBucket()

		Convey("Init works with no appState data", func() {
			err := init.FromGenesis(weave.Options{}, kv)
			So(err, ShouldBeNil)
		})

		Convey("Init works with no relevant appState data", func() {
			err := init.FromGenesis(weave.Options{"foo": []byte(`"bar"`)}, kv)
			So(err, ShouldBeNil)
		})

		Convey("Init fails with bad address", func() {
			err := init.FromGenesis(weave.Options{optKey: []byte(`{"addresses":["as"]}`)}, kv)
			So(err, ShouldNotBeNil)
		})

		Convey("Init succeeds with marshalled contents", func() {
			err := init.FromGenesis(weave.Options{optKey: accountsJson}, kv)
			So(err, ShouldBeNil)

			accounts, err := GetAccounts(bucket, kv)
			So(err, ShouldBeNil)
			So(accounts.Value(), ShouldResemble, AsAccounts(accts))
		})

		Convey("Init succeeds with hardcoded contents", func() {
			err := init.FromGenesis(weave.Options{optKey: accountsJson2}, kv)
			So(err, ShouldBeNil)

			accounts, err := GetAccounts(bucket, kv)
			So(err, ShouldBeNil)
			So(accounts.Value(), ShouldResemble, AsAccounts(accts2))
		})
	})
}
