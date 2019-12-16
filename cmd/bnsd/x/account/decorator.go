package account

import (
	weave "github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

// NewAccountMsgFeeDecorator returns a weave decorator that charge additional
// fee for each account modifying message as defined by the domain that
// modified account belongs to.
func NewAccountMsgFeeDecorator() weave.Decorator {
	return &accountMsgFeeDecorator{
		domains: NewDomainBucket(),
	}
}

type accountMsgFeeDecorator struct {
	domains orm.ModelBucket
}

func (d *accountMsgFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	if fee, err := d.msgFee(ctx, store, tx); err != nil {
		return nil, errors.Wrap(err, "msg fee")
	} else if fee != nil {
		// Adding ensures both values are the same currency.
		if sum, err := res.RequiredFee.Add(*fee); err != nil {
			return nil, errors.Wrap(err, "cannot merge fees")
		} else {
			res.RequiredFee = sum
		}
	}

	return res, nil
}

func (d *accountMsgFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	res, err := next.Deliver(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	if fee, err := d.msgFee(ctx, store, tx); err != nil {
		return nil, errors.Wrap(err, "msg fee")
	} else if fee != nil {
		// Adding ensures both values are the same currency.
		if sum, err := res.RequiredFee.Add(*fee); err != nil {
			return nil, errors.Wrap(err, "cannot merge fees")
		} else {
			res.RequiredFee = sum
		}
	}

	return res, nil
}

func (d *accountMsgFeeDecorator) msgFee(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*coin.Coin, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "get msg")
	}

	scopedMsg, ok := msg.(domainScopedMsg)
	if !ok {
		return nil, nil
	}

	var domain Domain
	switch err := d.domains.One(store, []byte(scopedMsg.GetDomain()), &domain); {
	case err == nil:
		// All good.
	case errors.ErrNotFound.Is(err):
		// If a domain does not exist, no extra price can be applied.
		return nil, nil
	default:
		return nil, errors.Wrap(err, "get domain")
	}

	for _, fee := range domain.MsgFees {
		if fee.MsgPath != msg.Path() {
			continue
		}

		return &fee.Fee, nil
	}
	return nil, nil
}

// domainScopedMsg is implemented by any weave.Msg that operates in context of
// a domain.
//
// This implementation depends on the fact that each protobuf message has
// GetXxx method generated for each attribute. When a message operates on a
// domain (or account), it will contain domain attribute.
type domainScopedMsg interface {
	GetDomain() string
}
