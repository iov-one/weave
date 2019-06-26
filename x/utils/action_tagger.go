package utils

import (
	"github.com/iov-one/weave"
	"github.com/tendermint/tendermint/libs/common"
)

// ActionTagger will inspect the message being executed and
// add a tag `action = msg.Path()`. This should be applied as
// a decorator so clients have a standard way to search / subscribe
// to eg. proposal creation.
//
// Note that for best results, this should be at the end of the
// ChainDecorators call, after Batch (so it is tagged with each submessage type).
// You will also want to wrap the governance router with this, so the result of
// a successful election will be tagged (and there won't be validators.ApplyDiffMsg
// being executed that do not show up in a search).
type ActionTagger struct{}

var _ weave.Decorator = ActionTagger{}

// ActionKey is used by ActionTagger as the Key in the Tag it appends
const ActionKey = "action"

// NewActionTagger creates a ActionTagger decorator
func NewActionTagger() ActionTagger {
	return ActionTagger{}
}

// Check just passes the request along
func (ActionTagger) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	return next.Check(ctx, db, tx)
}

// Deliver appends a tag on the result if there is a success.
func (ActionTagger) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	// if we error in reporting, let's do so early before dispatching
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	res, err := next.Deliver(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	tag := common.KVPair{
		Key:   []byte(ActionKey),
		Value: []byte(msg.Path()),
	}
	res.Tags = append(res.Tags, tag)
	return res, nil
}
