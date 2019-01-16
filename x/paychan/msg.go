package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/werrors"
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
		return werrors.New(werrors.InvalidMsg, "missing source")
	}
	if m.SenderPubkey == nil {
		return werrors.New(werrors.InvalidMsg, "missing sender public key")
	}
	if m.Recipient == nil {
		return werrors.New(werrors.InvalidMsg, "missing recipient")
	}
	if m.Total == nil || m.Total.IsZero() {
		return werrors.New(werrors.InvalidMsg, "inalid total amount")
	}
	if m.Timeout <= 0 {
		return werrors.New(werrors.InvalidMsg, "inalid timeout value")
	}
	if len(m.Memo) > 128 {
		return werrors.New(werrors.InvalidMsg, "memo too long")
	}

	return validateAddresses(m.Recipient, m.Src)
}

func (CreatePaymentChannelMsg) Path() string {
	return pathCreatePaymentChannelMsg
}

func (m *TransferPaymentChannelMsg) Validate() error {
	if m.Signature == nil {
		return werrors.New(werrors.InvalidMsg, "missing signature")
	}
	if m.Payment == nil {
		return werrors.New(werrors.InvalidMsg, "missing payment")
	}
	if m.Payment.ChainID == "" {
		return werrors.New(werrors.InvalidMsg, "missing chain ID")
	}
	if m.Payment.ChannelID == nil {
		return werrors.New(werrors.InvalidMsg, "missing channel ID")
	}
	if !m.Payment.Amount.IsPositive() {
		return werrors.New(werrors.InvalidMsg, "invalid amount value")
	}
	return nil
}

func (TransferPaymentChannelMsg) Path() string {
	return pathTransferPaymentChannelMsg
}

func (m *ClosePaymentChannelMsg) Validate() error {
	if m.ChannelID == nil {
		return werrors.New(werrors.InvalidMsg, "missing channel ID")
	}
	if len(m.Memo) > 128 {
		return werrors.New(werrors.InvalidMsg, "memo too long")
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
