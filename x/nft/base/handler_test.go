package base_test

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/base"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/username"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApprovalOpsHandler(t *testing.T) {
	Convey("Test approval ops handler", t, func() {
		ctx := context.Background()

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

		_ = d.Register(app.NftType_USERNAME.String(), userBucket)
		handler := base.NewApprovalOpsHandler(helpers.Authenticate(bob), nil, d)

		o, _ := chainBucket.Create(db, bob.Address(), chainId, nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
		chainBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobsUsername, nil, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, alice.Address(), aliceWithBobApproval, []nft.ActionApprovals{{
			Action:    nft.Action_ActionUpdateApprovals.String(),
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: 10, UntilBlockHeight: 5}, Address: bob.Address()}},
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
			msg := &nft.AddApprovalMsg{Id: bobsUsername,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateDetails.String(),
				Options: nft.ApprovalOptions{Count: nft.UnlimitedCount},
				T:       app.NftType_USERNAME.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobsUsername)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					owner := nft.FindActor(handler.Auth(), ctx, u, nft.Action_ActionUpdateApprovals.String())
					So(owner, ShouldResemble, bob.Address())
				})

				Convey("By approved", func() {
					msg.Address = guest.Address()
					msg.Id = aliceWithBobApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, aliceWithBobApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					usedApproval := getApproval(u.Approvals(), bob.Address(), nft.Action_ActionUpdateApprovals.String())
					So(usedApproval.Options.Count, ShouldEqual, 9)
					approved := nft.FindActor(handler.Auth(), ctx, u, nft.Action_ActionUpdateApprovals.String())
					So(approved, ShouldResemble, bob.Address())
					usedApproval = getApproval(u.Approvals(), bob.Address(), nft.Action_ActionUpdateApprovals.String())
					So(usedApproval.Options.Count, ShouldEqual, 8)
				})
			})

			Convey("Test error", func() {
				Convey("To owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Invalid", func() {
					tx := helpers.MockTx(msg)
					msg.Options.Count = 0
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = base.NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = bobsUsername
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.Address = alice.Address()
					msg.T = app.NftType_BLOCKCHAIN.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Exists", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					timeoutCtx := weave.WithHeight(ctx, 10)
					msg.Address = guest.Address()
					msg.Id = aliceWithBobApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(timeoutCtx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(timeoutCtx, db, tx)
					So(err, ShouldNotBeNil)
					o, err := userBucket.Get(db, aliceWithBobApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					approved := nft.FindActor(handler.Auth(), timeoutCtx, u, nft.Action_ActionUpdateApprovals.String())
					So(approved, ShouldBeNil)
				})
			})

		})

		Convey("Test Remove", func() {
			msg := &nft.RemoveApprovalMsg{Id: bobWithAliceApproval,
				Address: alice.Address(),
				Action:  nft.Action_ActionUpdateApprovals.String(),
				T:       app.NftType_USERNAME.String(),
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobWithAliceApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					owner := nft.FindActor(handler.Auth(), ctx, u, msg.Action)
					So(owner, ShouldResemble, bob.Address())
				})

				//TODO: Should we allow approved to remove their own approvals? :)
				Convey("By approved", func() {
					t.Logf("alice address: %s", alice.Address())
					handler = base.NewApprovalOpsHandler(helpers.Authenticate(alice), nil, d)
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobWithAliceApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					approved := nft.FindActor(handler.Auth(), ctx, u, msg.Action)
					So(approved, ShouldBeNil)
				})
			})

			Convey("Test error", func() {
				Convey("From owner", func() {
					tx := helpers.MockTx(msg)
					msg.Address = bob.Address()
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = base.NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					msg.Address = bob.Address()
					msg.Id = bobWithAliceApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown type", func() {
					msg.T = app.NftType_BLOCKCHAIN.String()
					msg.Id = chainId
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Does not exist", func() {
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.Id = []byte("123")
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Immutable", func() {
					msg.Id = bobWithAliceImmutableApproval
					tx := helpers.MockTx(msg)
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					msg.Id = bobWithAliceTimeoutApproval
					tx := helpers.MockTx(msg)
					timeoutCtx := weave.WithHeight(context.Background(), 10)
					handler = base.NewApprovalOpsHandler(helpers.Authenticate(guest), nil, d)
					_, err := handler.Check(timeoutCtx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(timeoutCtx, db, tx)
					So(err, ShouldNotBeNil)
				})
			})

		})

	})
}

func getApproval(approvals *nft.ApprovalOps, signer weave.Address, action string) nft.Approval {
	appr := approvals.
		List().
		ForAction(action).
		ForAddress(signer).
		AsPersistable()

	if len(appr) == 0 {
		panic("No approvals for action and user")
	}

	if len(appr[0].Approvals) == 0 {
		panic("No approvals for action and user")
	}

	return appr[0].Approvals[0]
}
