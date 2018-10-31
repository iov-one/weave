package base

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/username"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApprovalOpsHandler(t *testing.T) {
	Convey("Test approval ops handler", t, func() {
		bobsUsername := []byte("user")
		aliceWithBobApproval := []byte("user2")
		bobWithAliceApproval := []byte("user3")
		bobWithAliceImmutableApproval := []byte("user4")
		bobWithAliceTimeoutApproval := []byte("user5")

		chainId := []byte("any_network")
		var helpers x.TestHelpers
		_, alice := helpers.MakeKey()
		_, guest := helpers.MakeKey()
		_, bob := helpers.MakeKey()
		db := store.MemStore()
		chainBucket := blockchain.NewBucket()
		userBucket := username.NewBucket()
		d := nft.GetBucketDispatcher()

		_ = d.Register(app.NftType_Username.String(), userBucket)
		handler := NewApprovalOpsHandler(helpers.Authenticate(bob), nil, d)

		o, _ := chainBucket.Create(db, bob.Address(), chainId, nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
		chainBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobsUsername, nil, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, alice.Address(), aliceWithBobApproval, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceApproval, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceImmutableApproval, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, Immutable: true}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceTimeoutApproval, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, UntilBlockHeight: 5}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)

		Convey("Test add", func() {
			var blockHeight int64 = 5
			msg := &nft.AddApprovalMsg{Id: bobsUsername,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateDetails.String(),
				Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, UntilBlockHeight: 5},
				T:       app.NftType_Username.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobsUsername)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						FilterExpired(blockHeight).
						IsEmpty(), ShouldBeFalse)
				})

				Convey("By approved", func() {
					msg.Address = guest.Address()
					msg.Id = aliceWithBobApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, aliceWithBobApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						FilterExpired(blockHeight).
						IsEmpty(), ShouldBeFalse)
				})
			})

			Convey("Test error", func() {
				Convey("To owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Invalid", func() {
					tx := helpers.MockTx(msg)
					msg.Options.Count = 0
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = bobsUsername
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.Address = alice.Address()
					msg.T = app.NftType_Blockchain.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Exists", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					tx := helpers.MockTx(msg)
					var blockHeight int64 = 10
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobsUsername)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						FilterExpired(blockHeight).
						IsEmpty(), ShouldBeTrue)
				})
			})

		})

		Convey("Test Remove", func() {
			var blockHeight int64 = 5
			msg := &nft.RemoveApprovalMsg{Id: bobWithAliceApproval,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateApprovals.String(),
				T:       app.NftType_Username.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobWithAliceApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						FilterExpired(blockHeight).
						IsEmpty(), ShouldBeTrue)

				})

				//TODO: Should we allow approved to remove their own approvals? :)
				Convey("By approved", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(alice), nil, d)

					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobWithAliceApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						FilterExpired(blockHeight).
						IsEmpty(), ShouldBeTrue)
				})
			})

			Convey("Test error", func() {
				Convey("From owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = bobWithAliceApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.T = app.NftType_Blockchain.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Does not exist", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Immutable", func() {
					msg.Id = bobWithAliceImmutableApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					msg.Id = bobWithAliceTimeoutApproval
					tx := helpers.MockTx(msg)
					ctx := weave.WithHeight(context.Background(), 10)
					handler = NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})
			})

		})

	})
}
