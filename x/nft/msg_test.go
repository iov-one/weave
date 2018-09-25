package nft

import (
	"testing"

	"github.com/iov-one/weave/x"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApprovalMsg(t *testing.T) {
	Convey("Test add/remove approvals msg validation", t, func() {
		var helper x.TestHelpers
		_, validKey := helper.MakeKey()
		Convey("Test add approvals", func() {
			Convey("Happy flow", func() {
				msg := AddApprovalMsg{
					Id:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  ActionUpdateDetails,
				}
				Convey("Positive count", func() {
					msg.Options = ApprovalOptions{Count: 1}
					So(msg.Validate(), ShouldBeNil)
				})
				Convey("Unlimited count", func() {
					msg.Options = ApprovalOptions{Count: UnlimitedCount}
					So(msg.Validate(), ShouldBeNil)
				})
			})

			Convey("Testing various errors", func() {
				msg := AddApprovalMsg{
					Id:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  ActionUpdateDetails,
				}

				Convey("Invalid action", func() {
					msg.Action = "asd"
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid id", func() {
					msg.Id = []byte("asd")
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid address", func() {
					msg.Address = nil
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid count", func() {
					msg.Options = ApprovalOptions{Count: 0}
					So(msg.Validate(), ShouldNotBeNil)
				})
			})
		})

		Convey("Test Remove approvals", func() {
			Convey("Happy flow", func() {
				msg := RemoveApprovalMsg{
					Id:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  ActionUpdateDetails,
				}
				So(msg.Validate(), ShouldBeNil)
			})

			Convey("Testing various errors", func() {
				msg := RemoveApprovalMsg{
					Id:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  ActionUpdateDetails,
				}

				Convey("Invalid action", func() {
					msg.Action = "asd"
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid id", func() {
					msg.Id = []byte("asd")
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid address", func() {
					msg.Address = nil
					So(msg.Validate(), ShouldNotBeNil)
				})
			})
		})
	})
}
