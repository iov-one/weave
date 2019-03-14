package paychan

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
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
	bucket := NewPaymentChannelBucket()
	r.Handle(pathCreatePaymentChannelMsg, &createPaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
	r.Handle(pathTransferPaymentChannelMsg, &transferPaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
	r.Handle(pathClosePaymentChannelMsg, &closePaymentChannelHandler{auth: auth, bucket: bucket, cash: cash})
}

type createPaymentChannelHandler struct {
	auth   x.Authenticator
	bucket PaymentChannelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*createPaymentChannelHandler)(nil)

func (h *createPaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}

	res.GasAllocated += createPaymentChannelCost
	return res, nil
}

func (h *createPaymentChannelHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreatePaymentChannelMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*CreatePaymentChannelMsg)
	if !ok {
		return nil, errors.Wrap(errors.ErrInvalidMsg, "unknown transaction type")
	}

	if err := msg.Validate(); err != nil {
		return msg, err
	}

	// Ensure that the timeout is in the future.
	if height, _ := weave.GetHeight(ctx); msg.Timeout <= height {
		return msg, errors.Wrap(errors.ErrInvalidMsg, "timeout in the past")
	}

	if !h.auth.HasAddress(ctx, msg.Src) {
		return msg, errors.Wrap(errors.ErrUnauthorized, "invalid address")
	}

	return msg, nil
}

func (h *createPaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	obj, err := h.bucket.Create(db, &PaymentChannel{
		Src:          msg.Src,
		SenderPubkey: msg.SenderPubkey,
		Recipient:    msg.Recipient,
		Total:        msg.Total,
		Timeout:      msg.Timeout,
		Memo:         msg.Memo,
		Transferred:  &coin.Coin{Ticker: msg.Total.Ticker},
	})
	if err != nil {
		return res, errors.Wrap(err, "cannot create a payment channel")
	}

	// Move coins from sender account and deposit total amount available on
	// that channels account.
	dst := paymentChannelAccount(obj.Key())
	if err := h.cash.MoveCoins(db, msg.Src, dst, *msg.Total); err != nil {
		return res, errors.Wrap(err, "cannot move coins")
	}

	res.Data = obj.Key()
	return res, nil
}

type transferPaymentChannelHandler struct {
	auth   x.Authenticator
	bucket PaymentChannelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*transferPaymentChannelHandler)(nil)

func (h *transferPaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += transferPaymentChannelCost
	return res, nil
}

func (h *transferPaymentChannelHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TransferPaymentChannelMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get message")
	}
	msg, ok := rmsg.(*TransferPaymentChannelMsg)
	if !ok {
		return nil, errors.Wrap(errors.ErrInvalidMsg, "unknown tx type")
	}

	if err := msg.Validate(); err != nil {
		return msg, err
	}

	if weave.GetChainID(ctx) != msg.Payment.ChainID {
		return nil, errors.Wrap(errors.ErrInvalidMsg, "invalid chain ID")
	}

	pc, err := h.bucket.GetPaymentChannel(db, msg.Payment.ChannelID)
	if err != nil {
		return nil, err
	}

	// Check signature to ensure the message was not altered.
	raw, err := msg.Payment.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "cannot serialize payment")
	}
	if !pc.SenderPubkey.Verify(raw, msg.Signature) {
		return msg, errors.Wrap(errors.ErrInvalidMsg, "invalid signature")
	}

	if !msg.Payment.Amount.SameType(*pc.Total) {
		return msg, errors.Wrap(errors.ErrInvalidMsg, "amount and total amount use different ticker")
	}

	if msg.Payment.Amount.Compare(*pc.Total) > 0 {
		return msg, errors.Wrap(errors.ErrInvalidMsg, "amount greater than total amount")
	}
	// Payment is representing a cumulative amount that is to be
	// transferred to recipients account. Because it is cumulative, every
	// transfer request must be greater than the previous one.
	if msg.Payment.Amount.Compare(*pc.Transferred) <= 0 {
		return msg, errors.Wrap(errors.ErrInvalidMsg, "amount must be greater than previously requested")
	}

	return msg, nil
}

func (h *transferPaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	pc, err := h.bucket.GetPaymentChannel(db, msg.Payment.ChannelID)
	if err != nil {
		return res, err
	}

	// Payment amount is total amount that should be transferred from
	// payment channel to recipient. Deduct already transferred funds and
	// move only the difference.
	diff, err := msg.Payment.Amount.Subtract(*pc.Transferred)
	if err != nil || diff.IsZero() {
		return res, errors.Wrap(errors.ErrInvalidMsg, "invalid amount")
	}

	src := paymentChannelAccount(msg.Payment.ChannelID)
	if err := h.cash.MoveCoins(db, src, pc.Recipient, diff); err != nil {
		return res, err
	}

	// Track total amount transferred from the payment channel to the
	// recipients account.
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
		return res, err
	}

	obj := orm.NewSimpleObj(msg.Payment.ChannelID, pc)
	err = h.bucket.Save(db, obj)
	return res, err
}

type closePaymentChannelHandler struct {
	auth   x.Authenticator
	bucket PaymentChannelBucket
	cash   cash.Controller
}

var _ weave.Handler = (*closePaymentChannelHandler)(nil)

func (h *closePaymentChannelHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	return res, err
}

func (h *closePaymentChannelHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	pc, err := h.bucket.GetPaymentChannel(db, msg.ChannelID)
	if err != nil {
		return res, err
	}

	// If payment channel funds were exhausted anyone is free to close it.
	if pc.Total.Equals(*pc.Transferred) {
		err := h.bucket.Delete(db, msg.ChannelID)
		return res, err
	}

	if height, _ := weave.GetHeight(ctx); pc.Timeout > height {
		// If timeout was not reached, only the recipient is allowed to
		// close the channel.
		if !h.auth.HasAddress(ctx, pc.Recipient) {
			return res, errors.Wrap(errors.ErrInvalidMsg, "only the recipient is allowed to close the channel")
		}
	}

	// Before deleting the channel, return to sender all leftover funds
	// that are still allocated on this payment channel account.
	diff, err := pc.Total.Subtract(*pc.Transferred)
	if err != nil {
		return res, err
	}
	src := paymentChannelAccount(msg.ChannelID)
	if err := h.cash.MoveCoins(db, src, pc.Src, diff); err != nil {
		return res, err
	}
	err = h.bucket.Delete(db, msg.ChannelID)
	return res, err
}

func (h *closePaymentChannelHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ClosePaymentChannelMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*ClosePaymentChannelMsg)
	if !ok {
		return nil, errors.Wrap(errors.ErrInvalidMsg, "invalid message type")
	}

	return msg, msg.Validate()
}

// paymentChannelAccount returns an account address for a payment channel with
// given ID.
// Each payment channel deposit an initial value from sender to ensure that it
// is available to the recipient upon request. Each payment channel has a
// unique account address that can be deducted from its ID.
func paymentChannelAccount(paymentChannelId []byte) weave.Address {
	return weave.NewCondition("paychan", "seq", paymentChannelId).Address()
}
