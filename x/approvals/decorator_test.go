package approvals

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

var helpers x.TestHelpers

// newContextWithAuth creates a context with perms as signers and sets the height
func newContextWithAuth(perms ...weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
	return auth.SetConditions(ctx, perms...), auth
}

func TestSigDecorator(t *testing.T) {
	db := store.MemStore()
	tx := helpers.MockTx(helpers.MockMsg([]byte("test")))

	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()

	cases := []struct {
		signers []weave.Condition
		perms   []weave.Condition
	}{
		{[]weave.Condition{}, nil},
		{[]weave.Condition{a},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "usage")},
		},
		{[]weave.Condition{a, b},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "usage"),
				ApprovalCondition(b.Address(), "usage")}},
		{
			[]weave.Condition{a, b, c},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "usage"),
				ApprovalCondition(b.Address(), "usage"),
				ApprovalCondition(c.Address(), "usage"),
			},
		},
	}

	// the handler we're chaining with the decorator
	h := new(ApprovalCheckHandler)
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx, auth := newContextWithAuth(tc.signers...)
			d := NewSigDecorator(x.ChainAuth(auth, Authenticate{}))
			stack := helpers.Wrap(d, h)

			_, err := stack.Check(ctx, db, tx)
			require.NoError(t, err)
			assert.Equal(t, tc.perms, h.Perms)

			_, err = stack.Deliver(ctx, db, tx)
			require.NoError(t, err)
			assert.Equal(t, tc.perms, h.Perms)
		})
	}
}

//---------------- helpers --------

// MultisigCheckHandler stores the seen permissions on each call
// for this extension's authenticator (ie. multisig.Authenticate)
type ApprovalCheckHandler struct {
	Perms []weave.Condition
}

var _ weave.Handler = (*ApprovalCheckHandler)(nil)

func (s *ApprovalCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
}

func (s *ApprovalCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
}

func TestHasApproval(t *testing.T) {
	db := store.MemStore()
	tx := helpers.MockTx(helpers.MockMsg([]byte("test")))

	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()

	cases := []struct {
		signers []weave.Condition
		match   []weave.Condition
		not     []weave.Condition
	}{
		{
			[]weave.Condition{a},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "admin"),
				ApprovalCondition(a.Address(), "create"),
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(b.Address(), "admin"),
				ApprovalCondition(b.Address(), "create"),
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(b.Address(), "transfer"),
			},
		},
		{
			[]weave.Condition{b},
			[]weave.Condition{
				ApprovalCondition(b.Address(), "admin"),
				ApprovalCondition(b.Address(), "create"),
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(b.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "admin"),
				ApprovalCondition(a.Address(), "create"),
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
		},
	}

	// the handler we're chaining with the decorator
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx, auth := newContextWithAuth(tc.signers...)
			apprAuth := Authenticate{}
			h := &ApprovalCheckHandler{auth: apprAuth}
			multiauth := x.ChainAuth(auth, apprAuth)
			d := NewSigDecorator(multiauth)
			stack := helpers.Wrap(d, h)

			_, err := stack.Check(ctx, db, tx)
			require.NoError(t, err)

			_, err = stack.Deliver(ctx, db, tx)
			require.NoError(t, err)

			assert.True(t, HasApproval(ctx, multiauth, tc.match, "admin"))
			assert.True(t, HasApproval(ctx, multiauth, tc.match, "create"))
			assert.True(t, HasApproval(ctx, multiauth, tc.match, "update"))
			assert.True(t, HasApproval(ctx, multiauth, tc.match, "transfer"))

			assert.False(t, HasApproval(ctx, multiauth, tc.not, "admin"))
			assert.False(t, HasApproval(ctx, multiauth, tc.not, "create"))
			assert.False(t, HasApproval(ctx, multiauth, tc.not, "update"))
			assert.False(t, HasApproval(ctx, multiauth, tc.not, "transfer"))
		})
	}
}

// MultisigCheckHandler stores the seen permissions on each call
// for this extension's authenticator (ie. multisig.Authenticate)
type HasApprovalCheckHandler struct {
	actions    []string
	authorized []weave.Condition
	approved   []weave.Condition
}

var _ weave.Handler = (*HasApprovalCheckHandler)(nil)

func (s *HasApprovalCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	for _, action := range s.actions {
		HasApproval(ctx, Authenticate{}, s.authorized, action)
	}
	return
}

func (s *HasApprovalCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	for _, action := range s.actions {
		HasApproval(ctx, Authenticate{}, s.authorized, action)
	}
	return
}
