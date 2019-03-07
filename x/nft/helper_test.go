package nft

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFindActor(t *testing.T) {
	Convey("Test find actor", t, func() {
		ctx := context.Background()

		bob := weavetest.NewCondition()
		alice := weavetest.NewCondition()
		guest := weavetest.NewCondition()

		Convey("Test owner", func() {
			token := &NonFungibleToken{
				ID:    []byte("asd"),
				Owner: bob.Address(),
			}
			Convey("Test valid owner", func() {
				addr := FindActor(&weavetest.Auth{Signer: bob}, ctx, token, "")
				So(addr, ShouldResemble, bob.Address())
			})

			Convey("Test invalid owner", func() {
				addr := FindActor(&weavetest.Auth{Signer: alice}, ctx, token, "")
				So(addr, ShouldBeNil)
			})
		})

		Convey("Test approvals", func() {
			token := &NonFungibleToken{
				ID:    []byte("asd"),
				Owner: bob.Address(),
				ActionApprovals: []ActionApprovals{{
					Action:    UpdateDetails,
					Approvals: []Approval{{Options: ApprovalOptions{Count: UnlimitedCount, UntilBlockHeight: 5}, Address: alice.Address()}},
				}},
			}
			Convey("Test valid approval", func() {
				addr := FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateDetails)
				So(addr, ShouldResemble, alice.Address())
			})

			Convey("Test invalid action", func() {
				addr := FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateApprovals)
				So(addr, ShouldBeNil)
			})

			Convey("Test invalid signer", func() {
				addr := FindActor(&weavetest.Auth{Signer: guest}, ctx, token, "")
				So(addr, ShouldBeNil)
			})

			Convey("Test timeout", func() {
				addr := FindActor(&weavetest.Auth{Signer: guest}, weave.WithHeight(ctx, 10), token, "")
				So(addr, ShouldBeNil)
			})

			Convey("Test count decrements after use", func() {
				token.ActionApprovals = []ActionApprovals{{
					Action:    UpdateDetails,
					Approvals: []Approval{{Options: ApprovalOptions{Count: 10}, Address: alice.Address()}},
				}}

				addr := FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateDetails)
				So(addr, ShouldResemble, alice.Address())
				So(token.ActionApprovals[0].Approvals[0].Options.Count, ShouldEqual, 9)

				FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateDetails)
				So(token.ActionApprovals[0].Approvals[0].Options.Count, ShouldEqual, 8)
			})

			Convey("Test count decrements to 0 then disabled", func() {
				token.ActionApprovals = []ActionApprovals{{
					Action:    UpdateDetails,
					Approvals: []Approval{{Options: ApprovalOptions{Count: 1}, Address: alice.Address()}},
				}}

				addr := FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateDetails)
				So(addr, ShouldResemble, alice.Address())

				addr = FindActor(&weavetest.Auth{Signer: alice}, ctx, token, UpdateDetails)
				So(addr, ShouldBeNil)
			})
		})

	})
}
