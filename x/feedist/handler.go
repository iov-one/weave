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

// RegisterRoutes registers handlers for feedlist message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, ctrl CashController) {
	bucket := NewRevenueBucket()
	r.Handle(pathDistributeMsg, &distributeHandler{
		auth:   auth,
		bucket: bucket,
		ctrl:   ctrl,
	})
	r.Handle(pathUpdateRevenueMsg, &updateRevenueHandler{
		auth:   auth,
		bucket: bucket,
		ctrl:   ctrl,
	})
}

type distributeHandler struct {
	auth   x.Authenticator
	bucket *RevenueBucket
	ctrl   CashController
}

// CashController allows to manage coins stored by the accounts without the
// need to directly access the bucket.
// Required functionality is implemented by the x/cash extension.
type CashController interface {
	Balance(weave.KVStore, weave.Address) (x.Coins, error)
	MoveCoins(weave.KVStore, weave.Address, weave.Address, x.Coin) error
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

	if err := distribute(db, h.ctrl, weave.Address(msg.RevenueID), rev.Recipients); err != nil {
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
	if _, err := h.bucket.GetRevenue(db, msg.RevenueID); err != nil {
		return nil, errors.Wrap(err, "cannot get revenue")
	}
	return msg, nil
}

type updateRevenueHandler struct {
	auth   x.Authenticator
	bucket *RevenueBucket
	ctrl   CashController
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
	if err := distribute(db, h.ctrl, weave.Address(msg.RevenueID), rev.Recipients); err != nil {
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
//
// It might be that not all funds can be distributed equally. Because of that a
// small leftover can remain on the revenue account after this operation.
func distribute(db weave.KVStore, ctrl CashController, source weave.Address, recipients []*Recipient) error {
	var chunks int64
	for _, r := range recipients {
		chunks += int64(r.Weight)
	}

	balance, err := ctrl.Balance(db, source)
	switch {
	case err == nil:
		// All good.
	case errors.Is(errors.NotFoundErr, err):
		// Account does not exist, so there is are no funds to split.
		return nil
	default:
		return errors.Wrap(err, "cannot acquire revenue account balance")
	}

	// TODO normalize balance. There is no functionality that allows to
	// normalize x.Coins right now (14 Feb 2019).

	// For each currency, distribute the coins equally to the weight of
	// each recipient. This can leave small amount of coins on the original
	// account.
	for _, c := range balance {
		// Ignore those coins that have a negative value. This
		// functionality is supposed to be distributing value from
		// revenue account, not collect it. Otherwise this could be
		// used to charge the recipients instead of paying them.
		if !c.IsPositive() {
			continue
		}

		for _, r := range recipients {
			amount := x.Coin{
				Whole:      (c.Whole / chunks) * int64(r.Weight),
				Fractional: (c.Fractional / chunks) * int64(r.Weight),
				Ticker:     c.Ticker,
			}
			// Chunk is too small to be distributed.
			if amount.IsZero() {
				continue
			}
			if err := ctrl.MoveCoins(db, source, r.Address, amount); err != nil {
				return errors.Wrap(err, "cannot move coins")
			}
		}
	}

	return nil
}
