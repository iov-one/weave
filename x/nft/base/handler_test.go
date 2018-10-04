package base

import (
	"testing"

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
		userId := []byte("user")
		user2Id := []byte("user2")
		user3Id := []byte("user3")
		user4Id := []byte("user4")

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

		o, _ := chainBucket.Create(db, bob.Address(), chainId, nil, nil)
		chainBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), userId, nil, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, alice.Address(), user2Id, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), user3Id, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), user4Id, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, Immutable: true}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		cache := db.CacheWrap()

		Convey("Test add", func() {
			msg := &nft.AddApprovalMsg{Id: userId,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateDetails.String(),
				Options: nft.ApprovalOptions{Count: nft.UnlimitedCount},
				T:       app.NftType_Username.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(cache, userId)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						IsEmpty(), ShouldBeFalse)

				})

				Convey("By approved", func() {
					msg.Address = guest.Address()
					msg.Id = user2Id
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(cache, user2Id)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						IsEmpty(), ShouldBeFalse)
				})
			})

			Convey("Test error", func() {
				Convey("To owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Invalid", func() {
					tx := helpers.MockTx(msg)
					msg.Options.Count = 0
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = userId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.Address = alice.Address()
					msg.T = app.NftType_Blockchain.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Exists", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)

				})
			})

		})

		Convey("Test Remove", func() {
			msg := &nft.RemoveApprovalMsg{Id: user3Id,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateApprovals.String(),
				T:       app.NftType_Username.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(cache, user3Id)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						IsEmpty(), ShouldBeTrue)

				})

				//TODO: Should we allow approved to remove their own approvals? :)
				Convey("By approved", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(alice), nil, d)

					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(cache, user3Id)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					So(u.Approvals().
						List().
						ForAction(msg.Action).
						ForAddress(msg.Address).
						IsEmpty(), ShouldBeTrue)
				})
			})

			Convey("Test error", func() {
				Convey("From owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = user3Id
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.T = app.NftType_Blockchain.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Does not exist", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Immutable", func() {
					msg.Id = user4Id
					tx := helpers.MockTx(msg)
					_, err := handler.Check(nil, cache, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(nil, cache, tx)
					So(err, ShouldNotBeNil)
				})

			})

		})

	})
}
