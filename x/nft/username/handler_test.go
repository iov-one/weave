package username

import (
	"context"
	"testing"

	"github.com/iov-one/weave/errors"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/approvals"
	"github.com/iov-one/weave/x/nft"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/nft/blockchain"
	. "github.com/smartystreets/goconvey/convey"
)

var helpers x.TestHelpers

// newContextWithAuth creates a context with perms as signers and sets the height
func newContextWithAuth(perms ...weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
	return auth.SetConditions(ctx, perms...), auth
}

func TestHandlers(t *testing.T) {
	Convey("Test handlers", t, func() {
		db := store.MemStore()

		_, alice := helpers.MakeKey()
		_, bob := helpers.MakeKey()
		_, tom := helpers.MakeKey()

		tokenBucket := NewUsernameTokenBucket()
		blockchains := blockchain.NewBucket()

		for _, blockchainID := range []string{"myNet", "myOtherNet"} {
			b, _ := blockchains.Create(db, alice.Address(), []byte(blockchainID), nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
			blockchains.Save(db, b)
		}

		Convey("Test IssueHandler", func() {
			msg := IssueTokenMsg{
				Id:        []byte("second@example.com"),
				Owner:     alice.Address(),
				Approvals: [][]byte{approvals.ApprovalCondition(bob.Address(), "update")},
				Addresses: []*ChainAddress{{[]byte("myNet"), []byte("myNetAddress")}},
			}

			tx := helpers.MockTx(&msg)
			ctx, auth := newContextWithAuth(alice)
			h := IssueHandler{auth, nil, tokenBucket, blockchains}

			Convey("can create", func() {
				_, err := h.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = h.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
				token, _ := getUsernameToken(tokenBucket, db, msg.Id)
				So(*token, ShouldResemble, UsernameToken{msg.Id, msg.Owner, msg.Approvals, msg.Addresses})
			})

			Convey("id already exists", func() {
				_, err := h.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = h.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = h.Deliver(ctx, db, tx)
				So(err.Error(), ShouldEqual, orm.ErrUniqueConstraint("id exists already").Error())
			})

			Convey("invalid chain id", func() {
				msg.Addresses = []*ChainAddress{{[]byte("unknown"), []byte("unknown")}}
				tx := helpers.MockTx(&msg)
				_, err := h.Check(ctx, db, tx)
				So(err, ShouldNotBeNil)
				_, err = h.Deliver(ctx, db, tx)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("Test AddChainAddressMsgHandler", func() {
			tokenBucket.Save(db, orm.NewSimpleObj(
				[]byte("any1@example.com"),
				&UsernameToken{
					Id:        []byte("any1@example.com"),
					Owner:     alice.Address(),
					Approvals: [][]byte{approvals.ApprovalCondition(bob.Address(), "update")},
					Addresses: []*ChainAddress{{[]byte("myNet"), []byte("myNetAddress")}},
				}))

			msg := &AddChainAddressMsg{
				Id:        []byte("any1@example.com"),
				Addresses: &ChainAddress{[]byte("myOtherNet"), []byte("myOtherNetAddress")},
			}
			tx := helpers.MockTx(msg)

			Convey("owner can update", func() {
				ctx, auth := newContextWithAuth(alice)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := AddChainAddressHandler{multiauth, tokenBucket, blockchains}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
			})

			Convey("bob can update", func() {
				ctx, auth := newContextWithAuth(bob)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := AddChainAddressHandler{multiauth, tokenBucket, blockchains}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
			})

			Convey("tom cannot update", func() {
				ctx, auth := newContextWithAuth(tom)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := AddChainAddressHandler{multiauth, tokenBucket, blockchains}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err.Error(), ShouldEqual, errors.ErrUnauthorized().Error())
				_, err = stack.Deliver(ctx, db, tx)
				So(err.Error(), ShouldEqual, errors.ErrUnauthorized().Error())
			})

			Convey("chain duplicate", func() {
				msg.Addresses = &ChainAddress{[]byte("myNet"), []byte("myNetAddress")}
				tx := helpers.MockTx(msg)

				ctx, auth := newContextWithAuth(alice)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := AddChainAddressHandler{multiauth, tokenBucket, blockchains}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err.Error(), ShouldEqual, nft.ErrDuplicateEntry().Error())
			})
		})

		Convey("Test RemoveChainAddressHandler", func() {
			tokenBucket.Save(db, orm.NewSimpleObj(
				[]byte("any1@example.com"),
				&UsernameToken{
					Id:        []byte("any1@example.com"),
					Owner:     alice.Address(),
					Approvals: [][]byte{approvals.ApprovalCondition(bob.Address(), "update")},
					Addresses: []*ChainAddress{
						{[]byte("myNet"), []byte("myNetAddress")},
						{[]byte("myOtherNet"), []byte("myOtherChainAddress")},
					},
				}))

			msg := &RemoveChainAddressMsg{
				Id:        []byte("any1@example.com"),
				Addresses: &ChainAddress{[]byte("myNet"), []byte("myNetAddress")},
			}
			tx := helpers.MockTx(msg)

			Convey("owner can update", func() {
				ctx, auth := newContextWithAuth(alice)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := RemoveChainAddressHandler{multiauth, nil, tokenBucket}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
			})

			Convey("bob can update", func() {
				ctx, auth := newContextWithAuth(bob)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := RemoveChainAddressHandler{multiauth, nil, tokenBucket}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err, ShouldBeNil)
			})

			Convey("tom cannot update", func() {
				ctx, auth := newContextWithAuth(tom)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := RemoveChainAddressHandler{multiauth, nil, tokenBucket}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err.Error(), ShouldEqual, errors.ErrUnauthorized().Error())
				_, err = stack.Deliver(ctx, db, tx)
				So(err.Error(), ShouldEqual, errors.ErrUnauthorized().Error())
			})

			Convey("not found", func() {
				msg.Addresses = &ChainAddress{[]byte("unknown"), []byte("unknown")}
				tx := helpers.MockTx(msg)

				ctx, auth := newContextWithAuth(alice)
				multiauth := x.ChainAuth(auth, approvals.Authenticate{})
				h := RemoveChainAddressHandler{multiauth, nil, tokenBucket}
				d := approvals.NewDecorator(multiauth)
				stack := helpers.Wrap(d, h)
				_, err := stack.Check(ctx, db, tx)
				So(err, ShouldNotBeNil)
				_, err = stack.Deliver(ctx, db, tx)
				So(err, ShouldNotBeNil)
			})
		})
	})
}
