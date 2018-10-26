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
	if m.Sender == nil {
		return ErrMissingSender()
	}
	if m.SenderPublicKey == nil {
		return ErrMissingSenderPublicKey()
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

	return validateAddresses(m.Recipient, m.Sender)
}

func (CreatePaymentChannelMsg) Path() string {
	return pathCreatePaymentChannelMsg
}

func (m *TransferPaymentChannelMsg) Validate() error {
	panic("todo")
}

func (TransferPaymentChannelMsg) Path() string {
	return pathTransferPaymentChannelMsg
}

func (m *ClosePaymentChannelMsg) Validate() error {
	panic("todo")
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
