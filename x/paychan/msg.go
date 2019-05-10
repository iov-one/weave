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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
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
	if m.Timeout < inThePast {
		return errors.Wrap(errors.ErrInput, "timeout is in the past")
	}
	if err := m.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
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

// inThePast represents time value for Monday, January 1, 2018 2:00:00 AM GMT+01:00
//
// Assumption of this extension is that year 2018 is always in the past and it
// is safe to use as a broad border between the past and the future. This does
// not have to be a precise value as it should be used only for initial
// validation. Proper time validation must be done once the exact current time
// is available.
var inThePast weave.UnixTime = 1514768400
