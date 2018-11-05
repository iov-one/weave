package paychan

import (
	"github.com/iov-one/weave"
)

var _ weave.Msg = (*CreatePaymentChannelMsg)(nil)
var _ weave.Msg = (*TransferPaymentChannelMsg)(nil)
var _ weave.Msg = (*ClosePaymentChannelMsg)(nil)

const (
	pathCreatePaymentChannelMsg   = "paychan/create"
	pathTransferPaymentChannelMsg = "paychan/transfer"
	pathClosePaymentChannelMsg    = "paychan/close"
)

func (m *CreatePaymentChannelMsg) Validate() error {
	if m.Src == nil {
		return ErrMissingSrc()
	}
	if m.SenderPubkey == nil {
		return ErrMissingSenderPubkey()
	}
	if m.Recipient == nil {
		return ErrMissingRecipient()
	}
	if m.Total == nil || m.Total.IsZero() {
		return ErrInvalidTotal(m.Total)
	}
	if m.Timeout <= 0 {
		return ErrInvalidTimeout(m.Timeout)
	}
	if len(m.Memo) > 128 {
		return ErrInvalidMemo(m.Memo)
	}

	return validateAddresses(m.Recipient, m.Src)
}

func (CreatePaymentChannelMsg) Path() string {
	return pathCreatePaymentChannelMsg
}

func (m *TransferPaymentChannelMsg) Validate() error {
	if m.Signature == nil {
		return ErrMissingSignature()
	}
	if m.Payment == nil {
		return ErrMissingPayment()
	}
	if m.Payment.ChainId == "" {
		return ErrMissingChainID()
	}
	if m.Payment.ChannelId == nil {
		return ErrMissingChannelID()
	}
	if !m.Payment.Amount.IsPositive() {
		return ErrInvalidAmount(m.Payment.Amount)
	}
	return nil
}

func (TransferPaymentChannelMsg) Path() string {
	return pathTransferPaymentChannelMsg
}

func (m *ClosePaymentChannelMsg) Validate() error {
	if m.ChannelId == nil {
		return ErrMissingChannelID()
	}
	if len(m.Memo) > 128 {
		return ErrInvalidMemo(m.Memo)
	}
	return nil
}

func (ClosePaymentChannelMsg) Path() string {
	return pathClosePaymentChannelMsg
}

// validateAddresses returns an error if any non empty address does not
// validate.
func validateAddresses(addrs ...weave.Address) error {
	for _, a := range addrs {
		if a == nil {
			continue
		}
		if err := a.Validate(); err != nil {
			return err
		}
	}
	return nil
}
