package approvals

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
)

const UpdateAction = "update_approvals"

type AddApprovalMsgHandler struct {
	auth   x.Authenticator
	bucket orm.Bucket
}

func NewAddApprovalMsgHandler(auth x.Authenticator, bucket orm.Bucket) AddApprovalMsgHandler {
	return AddApprovalMsgHandler{auth, bucket}
}

var _ weave.Handler = AddApprovalMsgHandler{}

func (h AddApprovalMsgHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h AddApprovalMsgHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, appr, err := h.validate(ctx, store, tx)
	if err != nil {
		return res, err
	}

	ok := Approve(ctx, h.auth, UpdateAction, appr.GetApprovals(), appr.GetOwner())
	if !ok {
		return res, errors.ErrUnauthorized()
	}

	appr.UpdateApprovals(append(appr.GetApprovals(), msg.GetApproval()))
	obj := orm.NewSimpleObj(msg.GetId(), appr)
	err = h.bucket.Save(store, obj)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *AddApprovalMsgHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) (AddApprovalMsg, Approvable, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(AddApprovalMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(rmsg)
	}
	if err := validate(msg); err != nil {
		return nil, nil, err
	}
	appr, err := LoadApprovable(h.bucket, store, msg.GetId())
	if err != nil {
		return nil, nil, err
	}
	ok, _ = HasApprovals(ctx, h.auth, UpdateAction, appr.GetApprovals(), appr.GetOwner())
	if !ok {
		return nil, nil, errors.ErrUnauthorized()
	}
	return msg, appr, nil
}

// AsUsername will safely type-cast any value from Bucket
func AsApprovable(obj orm.Object) (Approvable, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(Approvable)
	if !ok {
		return nil, errors.ErrInternal("invalid id")
	}
	return x, nil
}

func LoadApprovable(bucket orm.Bucket, store weave.KVStore, id []byte) (Approvable, error) {
	o, err := bucket.Get(store, id)
	switch {
	case err != nil:
		return nil, err
	case o == nil:
		return nil, nft.ErrUnknownID()
	}
	t, e := AsApprovable(o)
	return t, e
}
