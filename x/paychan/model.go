package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*PaymentChannel)(nil)

// Validate ensures the payment channel is valid.
func (pc *PaymentChannel) Validate() error {
	if pc.Sender == nil {
		return ErrMissingSender()
	}
	if pc.SenderPublicKey == nil {
		return ErrMissingSenderPublicKey()
	}
	if pc.Recipient == nil {
		return ErrMissingRecipient()
	}
	if pc.Timeout <= 0 {
		return ErrInvalidTimeout(pc.Timeout)
	}
	if pc.Total == nil || !pc.Total.IsPositive() {
		return ErrInvalidTotal(pc.Total)
	}
	if len(pc.Memo) > 128 {
		return ErrInvalidMemo(pc.Memo)
	}

	// Transfer value must not be greater than the Total value represented
	// by the PaymentChannel.
	if pc.Transferred == nil || !pc.Transferred.IsPositive() || pc.Transferred.Compare(*pc.Total) > 0 {
		return ErrInvalidTransferred(pc.Transferred)
	}
	return nil
}

// Copy returns a shallow copy of this PaymentChannel.
func (pc PaymentChannel) Copy() orm.CloneableData {
	return &pc
}

// PaymentChannelBucket is a wrapper over orm.Bucket that ensures that only
// PaymentChannel entities can be persisted.
type PaymentChannelBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewPaymentChannelBucket returns a bucket for storing PaymentChannel state.
func NewPaymentChannelBucket() PaymentChannelBucket {
	b := orm.NewBucket("paychan", orm.NewSimpleObj(nil, &PaymentChannel{}))
	return PaymentChannelBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Create adds given payment store entity to the store and returns the ID of
// the newly inserted entity.
func (b *PaymentChannelBucket) Create(db weave.KVStore, pc *PaymentChannel) (orm.Object, error) {
	key := b.idSeq.NextVal(db)
	obj := orm.NewSimpleObj(key, pc)
	return obj, nil
}

// Save updates the state of given PaymentChannel entity in the store.
func (b *PaymentChannelBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*PaymentChannel); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

// GetPaymentChannel returns a payment channel instance with given ID or
// returns an error.
func (b *PaymentChannelBucket) GetPaymentChannel(db weave.KVStore, paymentChannelID []byte) (*PaymentChannel, error) {
	obj, err := b.Get(db, paymentChannelID)
	if err != nil {
		return nil, err
	}
	if obj == nil || obj.Value() == nil {
		return nil, ErrNoSuchPaymentChannel(paymentChannelID)
	}
	pc, ok := obj.Value().(*PaymentChannel)
	if !ok {
		return nil, ErrNoSuchPaymentChannel(paymentChannelID)
	}
	return pc, nil
}
