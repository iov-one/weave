package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r.Handle(&BumpSequenceMsg{}, migration.SchemaMigratingHandler("sigs",
		&bumpSequenceHandler{
			b:    NewBucket(),
			auth: auth,
		}))
}

type bumpSequenceHandler struct {
	auth x.Authenticator
	b    Bucket
}

func (h *bumpSequenceHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h *bumpSequenceHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	user, msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// Each transaction processing bumps the sequence by one. Increment
	// must represent the total increment value.
	incr := int64(msg.Increment) - 1
	if incr == 0 {
		// Zero increment requires no modification.
		return &weave.DeliverResult{}, nil
	}
	user.Sequence += incr
	obj := orm.NewSimpleObj(user.Pubkey.Address(), user)
	if err := h.b.Save(db, obj); err != nil {
		return nil, errors.Wrap(err, "save user")
	}

	return &weave.DeliverResult{}, nil
}

func (h *bumpSequenceHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UserData, *BumpSequenceMsg, error) {
	var msg BumpSequenceMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	pubkey := x.MainSigner(ctx, h.auth)
	if pubkey == nil {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "missing signature")
	}
	obj, err := h.b.Get(db, pubkey.Address())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bucket")
	}
	if obj == nil {
		return nil, nil, errors.Wrap(errors.ErrNotFound, "no sequence")
	}

	user := AsUser(obj)

	if user.Sequence+int64(msg.Increment) < user.Sequence {
		return nil, nil, errors.Wrap(errors.ErrOverflow, "user sequence")
	}

	return user, &msg, nil
}
