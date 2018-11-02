package approvals

import (
	"bytes"
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

func TestDecorator(t *testing.T) {
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
			d := NewDecorator(x.ChainAuth(auth, Authenticate{}))
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
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()

	cases := []struct {
		action     string
		signers    []weave.Condition
		authorized [][]byte
		match      []weave.Condition
		not        []weave.Condition
	}{
		{
			"update",
			[]weave.Condition{},
			[][]byte{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
			nil,
			[]weave.Condition{
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
				ApprovalCondition(b.Address(), "transfer"),
			},
		},
		{
			"update",
			[]weave.Condition{a},
			[][]byte{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "update"),
			},
			[]weave.Condition{
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
				ApprovalCondition(b.Address(), "transfer"),
			},
		},
		{
			"update",
			[]weave.Condition{a, b},
			[][]byte{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(b.Address(), "update"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "transfer"),
				ApprovalCondition(b.Address(), "transfer"),
			},
		},
		{
			"transfer",
			[]weave.Condition{a, b},
			[][]byte{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(a.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "transfer"),
			},
			[]weave.Condition{
				ApprovalCondition(a.Address(), "update"),
				ApprovalCondition(b.Address(), "update"),
				ApprovalCondition(b.Address(), "transfer"),
			},
		},
	}

	// the handler we're chaining with the decorator
	h := new(HasApprovalCheckHandler)
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx, auth := newContextWithAuth(tc.signers...)
			d := NewDecorator(x.ChainAuth(auth, Authenticate{}))
			stack := helpers.Wrap(d, h)

			msg := &nftMsg{helpers.MockMsg([]byte("nft")), tc.action, tc.authorized}
			tx := helpers.MockTx(msg)

			_, err := stack.Check(ctx, db, tx)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.match, h.approved)
			for _, a := range tc.not {
				assert.NotContains(t, h.approved, a)
			}

			_, err = stack.Deliver(ctx, db, tx)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.match, h.approved)
			for _, a := range tc.not {
				assert.NotContains(t, h.approved, a)
			}
		})
	}
}

// MultisigCheckHandler stores the seen permissions on each call
// for this extension's authenticator (ie. multisig.Authenticate)
type HasApprovalCheckHandler struct {
	approved []weave.Condition
}

var _ weave.Handler = (*HasApprovalCheckHandler)(nil)

func (s *HasApprovalCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	msg, _ := tx.GetMsg()
	nft := msg.(*nftMsg)
	var authorized []weave.Condition
	for _, auth := range nft.authorized {
		authorized = append(authorized, auth)
	}

	_, approved := HasApprovals(ctx, Authenticate{}, authorized, nft.action)
	s.approved = approved
	return
}

func (s *HasApprovalCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	msg, _ := tx.GetMsg()
	nft := msg.(*nftMsg)
	var authorized []weave.Condition
	for _, auth := range nft.authorized {
		authorized = append(authorized, auth)
	}

	_, approved := HasApprovals(ctx, Authenticate{}, authorized, nft.action)
	s.approved = approved
	return
}

// ContractTx fulfills the MultiSigTx interface to satisfy the decorator
type nftMsg struct {
	weave.Msg
	action     string
	authorized [][]byte
}

var _ weave.Msg = (*nftMsg)(nil)

func (nft nftMsg) Marshal() ([]byte, error) {
	bytes.Join(nft.authorized, []byte(","))
	return bytes.Join(nft.authorized, []byte(",")), nil
}

func (nft *nftMsg) Unmarshal(bz []byte) error {
	nft.authorized = bytes.Split(bz, []byte(","))
	return nil
}

func (nft nftMsg) Path() string {
	return "nft"
}

func (nft nftMsg) GetMsg() (weave.Msg, error) {
	return nft.Msg, nil
}
