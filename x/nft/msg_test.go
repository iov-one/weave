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
					ID:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  UpdateDetails,
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
					ID:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  UpdateDetails,
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
					ID:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  UpdateDetails,
				}
				So(msg.Validate(), ShouldBeNil)
			})

			Convey("Testing various errors", func() {
				msg := RemoveApprovalMsg{
					ID:      []byte("asdf"),
					Address: validKey.Address(),
					Action:  UpdateDetails,
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

func TestIsValidAction(t *testing.T) {
	cases := map[string]struct {
		action    interface{}
		wantValid bool
	}{
		"const declared action": {
			action:    UpdateDetails,
			wantValid: true,
		},
		"custom string action": {
			action:    string("ActionCatchPokemon"),
			wantValid: true,
		},
		"custom Action action": {
			action:    Action("ActionCatchPokemon"),
			wantValid: true,
		},
		"invalid action type": {
			action:    666,
			wantValid: false,
		},
		"action too short": {
			action:    "x",
			wantValid: false,
		},
		"action too long": {
			action:    "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			wantValid: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if isValidAction(tc.action) != tc.wantValid {
				t.Fatalf("want valid=%v", tc.wantValid)
			}
		})
	}
}
