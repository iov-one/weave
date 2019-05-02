package nft

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApprovalMsg(t *testing.T) {
	RegisterAction(DefaultActions...)

	Convey("Test add/remove approvals msg validation", t, func() {
		validKey := weavetest.NewCondition()
		Convey("Test add approvals", func() {
			Convey("Happy flow", func() {
				msg := AddApprovalMsg{
					Metadata: &weave.Metadata{Schema: 1},
					ID:       []byte("asdf"),
					Address:  validKey.Address(),
					Action:   UpdateDetails,
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
					Metadata: &weave.Metadata{Schema: 1},
					ID:       []byte("asdf"),
					Address:  validKey.Address(),
					Action:   UpdateDetails,
				}

				Convey("Invalid action", func() {
					msg.Action = "asd"
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid id", func() {
					msg.ID = []byte("asd")
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
					Metadata: &weave.Metadata{Schema: 1},
					ID:       []byte("asdf"),
					Address:  validKey.Address(),
					Action:   UpdateDetails,
				}
				So(msg.Validate(), ShouldBeNil)
			})

			Convey("Testing various errors", func() {
				msg := RemoveApprovalMsg{
					Metadata: &weave.Metadata{Schema: 1},
					ID:       []byte("asdf"),
					Address:  validKey.Address(),
					Action:   UpdateDetails,
				}

				Convey("Invalid action", func() {
					msg.Action = "asd"
					So(msg.Validate(), ShouldNotBeNil)
				})

				Convey("Invalid id", func() {
					msg.ID = []byte("as")
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
