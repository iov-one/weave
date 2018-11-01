package username

import (
	"context"
	"testing"

	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/approvals"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/nft/blockchain"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	Convey("Test approval ops handler", t, func() {
		db := store.MemStore()
		var helpers x.TestHelpers

		_, alice := helpers.MakeKey()
		_, bob := helpers.MakeKey()

		tokenBucket := NewUsernameTokenBucket()
		blockchains := blockchain.NewBucket()

		for _, blockchainID := range []string{"myNet", "myOtherNet"} {
			b, _ := blockchains.Create(db, alice.Address(), []byte(blockchainID), nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
			blockchains.Save(db, b)
		}

		tokenBucket.Save(db, orm.NewSimpleObj(
			[]byte("any1@example.com"),
			&UsernameToken{
				Id:        []byte("any1@example.com"),
				Owner:     alice.Address(),
				Approvals: [][]byte{approvals.ApprovalCondition(bob.Address(), "update")},
				Addresses: []*ChainAddress{{[]byte("myNet"), []byte("myNetAddress")}},
			}))

		Convey("Test update", func() {
			handler := AddChainAddressHandler{
				bucket:      tokenBucket,
				blockchains: blockchains,
			}
			msg := &AddChainAddressMsg{
				Id:        []byte("myNet"),
				Addresses: &ChainAddress{[]byte("myOtherNet"), []byte("myOtherNetAddress")},
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					handler.auth = helpers.Authenticate(alice)
					_, err := handler.Check(context.Background(), db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(context.Background(), db, tx)
					So(err, ShouldBeNil)
				})

				Convey("By approved", func() {
					tx := helpers.MockTx(msg)
					handler.auth = helpers.Authenticate(bob)
					_, err := handler.Check(context.Background(), db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(context.Background(), db, tx)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
