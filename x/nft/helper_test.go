package nft

import (
	"testing"

	"github.com/iov-one/weave/x"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFindActor(t *testing.T) {
	Convey("Test find actor", t, func() {
		var helper x.TestHelpers
		_, bob := helper.MakeKey()
		_, alice := helper.MakeKey()
		_, guest := helper.MakeKey()

		Convey("Test owner", func() {
			token := &NonFungibleToken{
				Id:    []byte("asd"),
				Owner: bob.Address(),
			}
			Convey("Test valid owner", func() {
				addr := FindActor(-1, helper.Authenticate(bob), nil, token, "")
				So(addr, ShouldResemble, bob.Address())
			})

			Convey("Test invalid owner", func() {
				addr := FindActor(-1, helper.Authenticate(alice), nil, token, "")
				So(addr, ShouldBeNil)
			})
		})

		Convey("Test approvals", func() {
			token := &NonFungibleToken{
				Id:    []byte("asd"),
				Owner: bob.Address(),
				ActionApprovals: []ActionApprovals{{
					Action:    Action_ActionUpdateDetails.String(),
					Approvals: []Approval{{Options: ApprovalOptions{Count: UnlimitedCount, UntilBlockHeight: 5}, Address: alice.Address()}},
				}},
			}
			Convey("Test valid approval", func() {
				addr := FindActor(-1, helper.Authenticate(alice), nil, token, Action_ActionUpdateDetails.String())
				So(addr, ShouldResemble, alice.Address())
			})

			Convey("Test invalid action", func() {
				addr := FindActor(-1, helper.Authenticate(alice), nil, token, Action_ActionUpdateApprovals.String())
				So(addr, ShouldBeNil)
			})

			Convey("Test invalid signer", func() {
				addr := FindActor(-1, helper.Authenticate(guest), nil, token, "")
				So(addr, ShouldBeNil)
			})

			Convey("Test timeout", func() {
				addr := FindActor(10, helper.Authenticate(guest), nil, token, "")
				So(addr, ShouldBeNil)
			})
		})

	})
}
