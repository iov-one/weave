package paychan

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

const (
	createPaymentChannelCost   int64 = 300
	transferPaymentChannelCost int64 = 5
)

// RegisterQuery registers payment channel bucket under /paychans.
func RegisterQuery(qr weave.QueryRouter) {
	NewPaymentChannelBucket().Register("paychans", qr)
}

// RegisterRouters registers payment channel message handelers in given registry.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, cash cash.Controller) {
	r = migration.SchemaMigratingRegistry("paychan", r)

	bucket := NewPaymentChannelBucket()
	r.Handle(&CreateMsg{},
		&createPaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
	r.Handle(&TransferMsg{},
		&transferPaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
	r.Handle(&CloseMsg{},
		&closePaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
}

type createPaymentChannelHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*createPaymentChannelHandler)(nil)

func (h *createPaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: createPaymentChannelCost}, nil
}

func (h *createPaymentChannelHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	if weave.IsExpired(ctx, msg.Timeout) {
		return nil, errors.Wrapf(errors.ErrExpired, "timeout in the past")
	}
	if !h.auth.HasAddress(ctx, msg.Source) {
		return &msg, errors.Wrap(errors.ErrUnauthorized, "invalid address")
	}
	return &msg, nil
}

func (h *createPaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// Explicitly acquire the ID as we need it before saving to compute
	// the entity address.
	key, err := paymentChannelSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire sequence ID")
	}

	pc := &PaymentChannel{
		Metadata:     &weave.Metadata{},
		Source:       msg.Source,
		SourcePubkey: msg.SourcePubkey,
		Destination:  msg.Destination,
		Total:        msg.Total,
		Timeout:      msg.Timeout,
		Memo:         msg.Memo,
		Transferred:  &coin.Coin{Ticker: msg.Total.Ticker},
		Address:      paymentChannelAccount(key),
	}
	if _, err := h.bucket.Put(db, key, pc); err != nil {
		return nil, errors.Wrap(err, "cannot create a payment channel")
	}

	// Move coins from source account and deposit total amount available on
	// that channels account.
	if err := h.cash.MoveCoins(db, msg.Source, pc.Address, *msg.Total); err != nil {
		return nil, errors.Wrap(err, "cannot move coins")
	}
	return &weave.DeliverResult{Data: key}, nil
}

// paymentChannelAccount returns an account address for a payment channel with
// given ID.
// Each payment channel deposit an initial value from source to ensure that it
// is available to the destination upon request. Each payment channel has a
// unique account address that can be deducted from its ID.
func paymentChannelAccount(paymentChannelId []byte) weave.Address {
	return weave.NewCondition("paychan", "seq", paymentChannelId).Address()
}

type transferPaymentChannelHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*transferPaymentChannelHandler)(nil)

func (h *transferPaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: transferPaymentChannelCost}, nil
}

func (h *transferPaymentChannelHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TransferMsg, error) {
	var msg TransferMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	if weave.GetChainID(ctx) != msg.Payment.ChainID {
		return nil, errors.Wrap(errors.ErrMsg, "invalid chain ID")
	}

	var pc PaymentChannel
	if err := h.bucket.One(db, msg.Payment.ChannelID, &pc); err != nil {
		return nil, err
	}

	// Check signature to ensure the message was not altered.
	raw, err := msg.Payment.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "cannot serialize payment")
	}
	if !pc.SourcePubkey.Verify(raw, msg.Signature) {
		return &msg, errors.Wrap(errors.ErrMsg, "invalid signature")
	}

	if !msg.Payment.Amount.SameType(*pc.Total) {
		return &msg, errors.Wrap(errors.ErrMsg, "amount and total amount use different ticker")
	}

	if msg.Payment.Amount.Compare(*pc.Total) > 0 {
		return &msg, errors.Wrap(errors.ErrMsg, "amount greater than total amount")
	}
	// Payment is representing a cumulative amount that is to be
	// transferred to destinations account. Because it is cumulative, every
	// transfer request must be greater than the previous one.
	if msg.Payment.Amount.Compare(*pc.Transferred) <= 0 {
		return &msg, errors.Wrap(errors.ErrMsg, "amount must be greater than previously requested")
	}

	return &msg, nil
}

func (h *transferPaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	var pc PaymentChannel
	if err := h.bucket.One(db, msg.Payment.ChannelID, &pc); err != nil {
		return nil, err
	}

	// Payment amount is total amount that should be transferred from
	// payment channel to destination. Deduct already transferred funds and
	// move only the difference.
	diff, err := msg.Payment.Amount.Subtract(*pc.Transferred)
	if err != nil || diff.IsZero() {
		return nil, errors.Wrap(errors.ErrMsg, "invalid amount")
	}

	if err := h.cash.MoveCoins(db, pc.Address, pc.Destination, diff); err != nil {
		return nil, err
	}

	// Track total amount transferred from the payment channel to the
	// destinations account.
	pc.Transferred = msg.Payment.Amount

	// We care about the latest memo only. Full history can be always
	// rebuild from the blockchain.
	pc.Memo = msg.Payment.Memo

	// If all funds were transferred, we can close the payment channel
	// because there is no further use for it. In addition, because all the
	// funds were used, no party is interested in closing it.
	//
	// To avoid "empty" payment channels in our database, delete it without
	// waiting for the explicit close request.
	if pc.Transferred.Equals(*pc.Total) {
		err := h.bucket.Delete(db, msg.Payment.ChannelID)
		return nil, err
	}

	if _, err := h.bucket.Put(db, msg.Payment.ChannelID, &pc); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

type closePaymentChannelHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*closePaymentChannelHandler)(nil)

func (h *closePaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	var msg CloseMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &weave.CheckResult{}, nil
}

func (h *closePaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	var msg CloseMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	var pc PaymentChannel
	if err := h.bucket.One(db, msg.ChannelID, &pc); err != nil {
		return nil, err
	}

	// If payment channel funds were exhausted anyone is free to close it.
	if pc.Total.Equals(*pc.Transferred) {
		err := h.bucket.Delete(db, msg.ChannelID)
		return nil, err
	}

	if !weave.IsExpired(ctx, pc.Timeout) {
		// If timeout was not reached, only the destination is allowed to
		// close the channel.
		if !h.auth.HasAddress(ctx, pc.Destination) {
			return nil, errors.Wrap(errors.ErrMsg, "only the destination is allowed to close the channel")
		}
	}

	// Before deleting the channel, return to source all leftover funds
	// that are still allocated on this payment channel account.
	diff, err := pc.Total.Subtract(*pc.Transferred)
	if err != nil {
		return nil, err
	}
	if err := h.cash.MoveCoins(db, pc.Address, pc.Source, diff); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, msg.ChannelID); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}
