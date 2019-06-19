package paychan

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &PaymentChannel{}, migration.NoModification)
}

var _ orm.CloneableData = (*PaymentChannel)(nil)

// Validate ensures the payment channel is valid.
func (pc *PaymentChannel) Validate() error {
	if err := pc.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := pc.Src.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if pc.SenderPubkey == nil {
		return errors.Wrap(errors.ErrModel, "missing sender public key")
	}
	if err := pc.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if pc.Timeout < inThePast {
		return errors.Wrap(errors.ErrInput, "timeout is required")
	}
	if err := pc.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if pc.Total == nil || !pc.Total.IsPositive() {
		return errors.Wrap(errors.ErrModel, "negative total")
	}
	if len(pc.Memo) > 128 {
		return errors.Wrap(errors.ErrModel, "memo too long")
	}

	// Transfer value must not be greater than the Total value represented
	// by the PaymentChannel.
	if pc.Transferred == nil || !pc.Transferred.IsNonNegative() || pc.Transferred.Compare(*pc.Total) > 0 {
		return errors.Wrap(errors.ErrModel, "invalid transferred value")
	}
	return nil
}

// Copy returns a deep copy of this PaymentChannel.
func (pc PaymentChannel) Copy() orm.CloneableData {
	return &PaymentChannel{
		Metadata:     pc.Metadata.Copy(),
		Src:          pc.Src.Clone(),
		SenderPubkey: pc.SenderPubkey,
		Recipient:    pc.Recipient.Clone(),
		Total:        pc.Total.Clone(),
		Timeout:      pc.Timeout,
		Memo:         pc.Memo,
		Transferred:  pc.Transferred.Clone(),
	}
}

// NewPaymentChannelBucket returns a bucket for storing PaymentChannel state.
func NewPaymentChannelBucket() orm.XModelBucket {
	b := orm.NewModelBucket("paychan", &PaymentChannel{})
	return migration.NewModelBucket("paychan", b)
}

func newPaymentChannelObjectBucket() orm.BaseBucket {
	obj := orm.NewSimpleObj(nil, &PaymentChannel{})
	return orm.NewBucket("paychan", obj)
}
