package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

func RegisterRoutes(r weave.Registry) {
	r.Handle(pathBumpSequenceMsg, &bumpSequenceHandler{
		b: NewBucket(),
	})
}

type bumpSequenceHandler struct {
	b Bucket
}

func (h *bumpSequenceHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	return weave.CheckResult{}, err
}

func (h *bumpSequenceHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	user, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return weave.DeliverResult{}, err
	}

	user.Sequence += msg.Increment
	obj := orm.NewSimpleObj(user.Pubkey.Address(), user)
	if err := h.b.Save(db, obj); err != nil {
		return weave.DeliverResult{}, errors.Wrap(err, "save user")
	}

	return weave.DeliverResult{}, nil
}

func (h *bumpSequenceHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UserData, *BumpSequenceMsg, error) {
	var msg BumpSequenceMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	obj, err := h.b.Get(db, msg.Pubkey.Address())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bucket")
	}
	if obj == nil {
		return nil, nil, errors.Wrap(errors.ErrNotFound, "unknown public key")
	}

	return AsUser(obj), &msg, nil
}
