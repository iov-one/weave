package approvals

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/assert"
)

func TestApprovalCondition(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	cond := ApprovalCondition(alice.Address(), "update")

	err := cond.Validate()
	assert.NoError(t, err)

	_, action, id, err := cond.Parse()
	assert.NoError(t, err)

	assert.Equal(t, action, "update")
	assert.Equal(t, alice.Address(), weave.Address(id))
}

func TestContext(t *testing.T) {
	// sig is a signature permission for contractID, not a contract ID
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	bg := context.Background()
	cases := []struct {
		action string
		ctx    weave.Context
		match  [][]byte
		not    [][]byte
	}{
		{
			"update",
			withApproval(bg, alice.Address()),
			[][]byte{
				ApprovalCondition(alice.Address(), "update"),
			},
			[][]byte{
				ApprovalCondition(alice.Address(), "create"),
				ApprovalCondition(bob.Address(), "update"),
			},
		},
		{
			"create",
			withApproval(bg, alice.Address()),
			[][]byte{
				ApprovalCondition(alice.Address(), "create"),
			},
			[][]byte{
				ApprovalCondition(alice.Address(), "update"),
				ApprovalCondition(bob.Address(), "update"),
			},
		},
	}

	auth := Authenticate{}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ok, _ := HasApprovals(tc.ctx, auth, tc.action, tc.match)
			assert.True(t, ok)

			ok, _ = HasApprovals(tc.ctx, auth, tc.action, tc.not)
			assert.False(t, ok)
		})
	}
}
