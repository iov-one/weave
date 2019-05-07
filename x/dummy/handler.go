package dummy

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func RegisterRoutes(r weave.Registry) {
	bucket := NewCartonBoxBucket()
	r.Handle(CreateCartonBoxMsg{}.Path(), migration.SchemaMigratingHandler("dummy", &createCartonBoxHandler{
		bucket: bucket,
	}))
	r.Handle(InspectCartonBoxMsg{}.Path(), migration.SchemaMigratingHandler("dummy", &inspectCartonBoxHandler{
		bucket: bucket,
	}))

}

type createCartonBoxHandler struct {
	bucket *CartonBoxBucket
}

func (h *createCartonBoxHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h *createCartonBoxHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	obj, err := h.bucket.Create(db, &CartonBox{
		Metadata: &weave.Metadata{},
		Width:    msg.Width,
		Height:   msg.Height,
		Quality:  msg.Quality,
	})
	if err != nil {
		return nil, err
	}
	return &weave.DeliverResult{Data: obj.Key()}, nil
}

func (h *createCartonBoxHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateCartonBoxMsg, error) {
	var msg CreateCartonBoxMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}

type inspectCartonBoxHandler struct {
	bucket *CartonBoxBucket
}

func (h *inspectCartonBoxHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h *inspectCartonBoxHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	cbox, err := h.bucket.CartonBoxByID(db, msg.CartonBoxID)
	if err != nil {
		return nil, errors.Wrap(err, "bucket get by ID")
	}
	raw, err := cbox.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal CartonBox")
	}
	return &weave.DeliverResult{Data: raw}, nil
}

func (h *inspectCartonBoxHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*InspectCartonBoxMsg, error) {
	var msg InspectCartonBoxMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}

func RegisterQuery(qr weave.QueryRouter) {
	NewCartonBoxBucket().Register("cartonboxes", qr)
}
