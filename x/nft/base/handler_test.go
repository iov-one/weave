package base_test

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/base"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApprovalOpsHandler(t *testing.T) {
	nft.RegisterAction(nft.DefaultActions...)

	Convey("Test approval ops handler", t, func() {
		ctx := context.Background()

		bobsUsername := []byte("user")
		aliceWithBobApproval := []byte("user2")
		bobWithAliceApproval := []byte("user3")
		bobWithAliceImmutableApproval := []byte("user4")
		bobWithAliceTimeoutApproval := []byte("user5")

		alice := weavetest.NewCondition()
		guest := weavetest.NewCondition()
		bob := weavetest.NewCondition()
		db := store.MemStore()
		userBucket := username.NewBucket()
		nftBuckets := map[string]orm.Bucket{
			username.ModelName: userBucket.Bucket,
		}

		handler := base.NewApprovalOpsHandler(&weavetest.Auth{Signer: bob}, nil, nftBuckets)

		o, _ := userBucket.Create(db, bob.Address(), bobsUsername, nil, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, alice.Address(), aliceWithBobApproval, []nft.ActionApprovals{{
			Action:    nft.UpdateApprovals,
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: 10, UntilBlockHeight: 5}, Address: bob.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceApproval, []nft.ActionApprovals{{
			Action:    nft.UpdateApprovals,
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceImmutableApproval, []nft.ActionApprovals{{
			Action:    nft.UpdateApprovals,
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, Immutable: true}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)
		o, _ = userBucket.Create(db, bob.Address(), bobWithAliceTimeoutApproval, []nft.ActionApprovals{{
			Action:    nft.UpdateApprovals,
			Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount, UntilBlockHeight: 5}, Address: alice.Address()}},
		}}, nil)
		userBucket.Save(db, o)

		Convey("Test add", func() {
			msg := &nft.AddApprovalMsg{ID: bobsUsername,
				Address: alice.Address(),
				Action:  nft.UpdateDetails,
				Options: nft.ApprovalOptions{Count: nft.UnlimitedCount},
				T:       username.ModelName,
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, bobsUsername)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					owner := nft.FindActor(handler.Auth(), ctx, u, nft.UpdateApprovals)
					So(owner, ShouldResemble, bob.Address())
				})

				Convey("By approved", func() {
					msg.Address = guest.Address()
					msg.ID = aliceWithBobApproval
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					o, err := userBucket.Get(db, aliceWithBobApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					usedApproval := getApproval(u.Approvals(), bob.Address(), nft.UpdateApprovals)
					So(usedApproval.Options.Count, ShouldEqual, 9)
					approved := nft.FindActor(handler.Auth(), ctx, u, nft.UpdateApprovals)
					So(approved, ShouldResemble, bob.Address())
					usedApproval = getApproval(u.Approvals(), bob.Address(), nft.UpdateApprovals)
					So(usedApproval.Options.Count, ShouldEqual, 8)
				})
			})

			Convey("Test error", func() {
				Convey("To owner", func() {
					tx := &weavetest.Tx{Msg: msg}
					msg.Address = bob.Address()
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Invalid", func() {
					tx := &weavetest.Tx{Msg: msg}
					msg.Options.Count = 0
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = base.NewApprovalOpsHandler(&weavetest.Auth{Signer: guest}, nil, nftBuckets)
					msg.Address = bob.Address()
					msg.ID = bobsUsername
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Exists", func() {
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.ID = []byte("123")
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					timeoutCtx := weave.WithHeight(ctx, 10)
					msg.Address = guest.Address()
					msg.ID = aliceWithBobApproval
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(timeoutCtx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(timeoutCtx, db, tx)
					So(err, ShouldNotBeNil)
					o, err := userBucket.Get(db, aliceWithBobApproval)
					So(err, ShouldBeNil)
					u, err := username.AsUsername(o)
					So(err, ShouldBeNil)
					approved := nft.FindActor(handler.Auth(), timeoutCtx, u, nft.UpdateApprovals)
					So(approved, ShouldBeNil)
				})
			})

		})

		Convey("Test Remove", func() {
			msg := &nft.RemoveApprovalMsg{ID: bobWithAliceApproval,
				Address: alice.Address(),
				Action:  nft.UpdateApprovals,
				T:       username.ModelName,
			}
			Convey("Test happy", func() {
				Convey("By owner", func() {
					tx := &weavetest.Tx{Msg: msg}
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
					handler = base.NewApprovalOpsHandler(&weavetest.Auth{Signer: alice}, nil, nftBuckets)
					tx := &weavetest.Tx{Msg: msg}
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
					tx := &weavetest.Tx{Msg: msg}
					msg.Address = bob.Address()
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("By guest", func() {
					handler = base.NewApprovalOpsHandler(&weavetest.Auth{Signer: guest}, nil, nftBuckets)
					msg.Address = bob.Address()
					msg.ID = bobWithAliceApproval
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Does not exist", func() {
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Unknown id", func() {
					msg.ID = []byte("123")
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Immutable", func() {
					msg.ID = bobWithAliceImmutableApproval
					tx := &weavetest.Tx{Msg: msg}
					_, err := handler.Check(ctx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(ctx, db, tx)
					So(err, ShouldNotBeNil)
				})

				Convey("Timeout", func() {
					msg.ID = bobWithAliceTimeoutApproval
					tx := &weavetest.Tx{Msg: msg}
					timeoutCtx := weave.WithHeight(context.Background(), 10)
					handler = base.NewApprovalOpsHandler(&weavetest.Auth{Signer: guest}, nil, nftBuckets)
					_, err := handler.Check(timeoutCtx, db, tx)
					So(err, ShouldBeNil)
					_, err = handler.Deliver(timeoutCtx, db, tx)
					So(err, ShouldNotBeNil)
				})
			})

		})

	})
}

func getApproval(approvals *nft.ApprovalOps, signer weave.Address, action nft.Action) nft.Approval {
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
