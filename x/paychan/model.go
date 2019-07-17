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
	var errs error

	errs = errors.AppendField(errs, "Metadata", pc.Metadata.Validate())
	errs = errors.AppendField(errs, "Source", pc.Source.Validate())
	if pc.SourcePubkey == nil {
		errs = errors.Append(errs,
			errors.Field("SourcePubKey", errors.ErrModel, "missing source public key"))
	}
	errs = errors.AppendField(errs, "destination", pc.Destination.Validate())
	if err := pc.Timeout.Validate(); err != nil {
		errs = errors.AppendField(errs, "Timeout", err)
	} else if pc.Timeout < inThePast {
		errs = errors.Append(errs,
			errors.Field("Timeout", errors.ErrInput, "timeout is required"))
	}
	if pc.Total == nil || !pc.Total.IsPositive() {
		errs = errors.Append(errs,
			errors.Field("Total", errors.ErrModel, "negative total"))
	}
	if len(pc.Memo) > 128 {
		errs = errors.Append(errs,
			errors.Field("Memo", errors.ErrModel, "memo too long"))
	}

	// Transfer value must not be greater than the Total value represented
	// by the PaymentChannel.
	if pc.Transferred == nil || !pc.Transferred.IsNonNegative() || pc.Transferred.Compare(*pc.Total) > 0 {
		errs = errors.Append(errs,
			errors.Field("Transferred", errors.ErrModel, "invalid transferred value"))
	}

	if err := pc.Address.Validate(); err != nil {
		errs = errors.AppendField(errs, "Address", err)
	}

	return errs
}

// Copy returns a deep copy of this PaymentChannel.
func (pc PaymentChannel) Copy() orm.CloneableData {
	return &PaymentChannel{
		Metadata:     pc.Metadata.Copy(),
		Source:       pc.Source.Clone(),
		SourcePubkey: pc.SourcePubkey,
		Destination:  pc.Destination.Clone(),
		Total:        pc.Total.Clone(),
		Timeout:      pc.Timeout,
		Memo:         pc.Memo,
		Transferred:  pc.Transferred.Clone(),
	}
}

// NewPaymentChannelBucket returns a bucket for storing PaymentChannel state.
func NewPaymentChannelBucket() orm.ModelBucket {
	b := orm.NewModelBucket("paychan", &PaymentChannel{},
		orm.WithIDSequence(paymentChannelSeq))
	return migration.NewModelBucket("paychan", b)
}

var paymentChannelSeq = orm.NewSequence("paychan", "id")

func newPaymentChannelObjectBucket() orm.Bucket {
	obj := orm.NewSimpleObj(nil, &PaymentChannel{})
	return orm.NewBucket("paychan", obj)
}
