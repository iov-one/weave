package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

var _ weave.Msg = (*CreatePaymentChannelMsg)(nil)
var _ weave.Msg = (*TransferPaymentChannelMsg)(nil)
var _ weave.Msg = (*ClosePaymentChannelMsg)(nil)

func init() {
	migration.MustRegister(1, &CreatePaymentChannelMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferPaymentChannelMsg{}, migration.NoModification)
	migration.MustRegister(1, &ClosePaymentChannelMsg{}, migration.NoModification)
}

const (
	pathCreatePaymentChannelMsg   = "paychan/create"
	pathTransferPaymentChannelMsg = "paychan/transfer"
	pathClosePaymentChannelMsg    = "paychan/close"
)

func (m *CreatePaymentChannelMsg) Validate() error {
	if m.Src == nil {
		return errors.Wrap(errors.ErrMsg, "missing source")
	}
	if m.SenderPubkey == nil {
		return errors.Wrap(errors.ErrMsg, "missing sender public key")
	}
	if m.Recipient == nil {
		return errors.Wrap(errors.ErrMsg, "missing recipient")
	}
	if m.Total == nil || m.Total.IsZero() {
		return errors.Wrap(errors.ErrMsg, "invalid total amount")
	}
	if m.Timeout <= 0 {
		return errors.Wrap(errors.ErrMsg, "invalid timeout value")
	}
	if len(m.Memo) > 128 {
		return errors.Wrap(errors.ErrMsg, "memo too long")
	}

	return validateAddresses(m.Recipient, m.Src)
}

func (CreatePaymentChannelMsg) Path() string {
	return pathCreatePaymentChannelMsg
}

func (m *TransferPaymentChannelMsg) Validate() error {
	if m.Signature == nil {
		return errors.Wrap(errors.ErrMsg, "missing signature")
	}
	if m.Payment == nil {
		return errors.Wrap(errors.ErrMsg, "missing payment")
	}
	if m.Payment.ChainID == "" {
		return errors.Wrap(errors.ErrMsg, "missing chain ID")
	}
	if m.Payment.ChannelID == nil {
		return errors.Wrap(errors.ErrMsg, "missing channel ID")
	}
	if !m.Payment.Amount.IsPositive() {
		return errors.Wrap(errors.ErrMsg, "invalid amount value")
	}
	return nil
}

func (TransferPaymentChannelMsg) Path() string {
	return pathTransferPaymentChannelMsg
}

func (m *ClosePaymentChannelMsg) Validate() error {
	if m.ChannelID == nil {
		return errors.Wrap(errors.ErrMsg, "missing channel ID")
	}
	if len(m.Memo) > 128 {
		return errors.Wrap(errors.ErrMsg, "memo too long")
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
