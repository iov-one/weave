package feedist

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	distributeCost    = 0
	updateRevenueCost = 0
)

// RegisterQuery registers feedlist buckets for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewRevenueBucket().Register("revenues", qr)
}

// RegisterRouter registers handlers for feedlist message processing.
func RegisterRouter(r weave.Registry, auth x.Authenticator) {
	bucket := NewRevenueBucket()
	r.Handle(pathDistributeMsg, &distributeHandler{auth: auth, bucket: bucket})
	r.Handle(pathUpdateRevenueMsg, &updateRevenueHandler{auth: auth, bucket: bucket})
}

type distributeHandler struct {
	auth   x.Authenticator
	bucket *RevenueBucket
}

func (h *distributeHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += distributeCost
	return res, nil
}

func (h *distributeHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	rev, err := h.bucket.GetRevenue(db, msg.RevenueID)
	if err != nil {
		return res, err
	}

	if err := distribute(ctx, db, rev); err != nil {
		return res, errors.Wrap(err, "cannot distribute")
	}
	return res, nil
}

func (h *distributeHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DistributeMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get message")
	}
	msg, ok := rmsg.(*DistributeMsg)
	if !ok {
		return nil, errors.InvalidMsgErr.New("unknown transaction type")
	}
	if err := msg.Validate(); err != nil {
		return msg, err
	}
	return msg, nil
}

type updateRevenueHandler struct {
	auth   x.Authenticator
	bucket *RevenueBucket
}

func (h *updateRevenueHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += updateRevenueCost
	return res, nil
}

func (h *updateRevenueHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	rev, err := h.bucket.GetRevenue(db, msg.RevenueID)
	if err != nil {
		return res, err
	}

	// Before updating the revenue all funds must be distributed. Only a
	// revenue with no funds can be updated, so that recipients trust us.
	// Otherwise an admin could change who receives the money without the
	// previously selected recepients ever being paid.
	if err := distribute(ctx, db, rev); err != nil {
		return res, errors.Wrap(err, "cannot distribute")
	}

	rev.Recipients = msg.Recipients
	obj := orm.NewSimpleObj(msg.RevenueID, rev)
	if err := h.bucket.Save(db, obj); err != nil {
		return res, errors.Wrap(err, "cannot save")
	}
	return res, nil
}

func (h *updateRevenueHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateRevenueMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get message")
	}
	msg, ok := rmsg.(*UpdateRevenueMsg)
	if !ok {
		return nil, errors.InvalidMsgErr.New("unknown transaction type")
	}
	if err := msg.Validate(); err != nil {
		return msg, err
	}
	return msg, nil
}

// distribute split the funds stored under the revenue address and distribute
// them according to recipients proportions. When successful, revenue account
// has no funds left after this call.
func distribute(ctx weave.Context, db weave.KVStore, rev *Revenue) error {
	var chunks int32
	for _, r := range rev.Recipients {
		chunks += r.Weight
	}

	panic("todo")
}
