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
	var errs error

	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Src", m.Src.Validate())
	if m.SenderPubkey == nil {
		errs = errors.Append(errs,
			errors.Field("SenderPubKey", errors.ErrMsg, "missing sender public key"))
	}
	errs = errors.AppendField(errs, "Recipient", m.Recipient.Validate())
	if err := m.Timeout.Validate(); err != nil {
		errs = errors.AppendField(errs, "Timeout", err)
	} else if m.Timeout < inThePast {
		errs = errors.Append(errs,
			errors.Field("Timeout", errors.ErrInput, "timeout is required"))
	}
	if m.Total == nil || !m.Total.IsPositive() {
		errs = errors.Append(errs,
			errors.Field("Total", errors.ErrMsg, "negative total"))
	}
	if len(m.Memo) > 128 {
		errs = errors.Append(errs,
			errors.Field("Memo", errors.ErrMsg, "memo too long"))
	}
	return errs
}

func (CreatePaymentChannelMsg) Path() string {
	return pathCreatePaymentChannelMsg
}

func (m *TransferPaymentChannelMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if m.Signature == nil {
		errs = errors.Append(errs,
			errors.Field("Signature", errors.ErrMsg, "missing signature"))
	}
	if m.Payment == nil {
		errs = errors.Append(errs,
			errors.Field("Payment", errors.ErrMsg, "missing payment"))
	} else {
		if m.Payment.ChainID == "" {
			errs = errors.Append(errs,
				errors.Field("Payment.ChainID", errors.ErrMsg, "missing chain ID"))
		}
		if m.Payment.ChannelID == nil {
			errs = errors.Append(errs,
				errors.Field("Payment.ChannelID", errors.ErrMsg, "missing channel ID"))
		}
		if !m.Payment.Amount.IsPositive() {
			errs = errors.Append(errs,
				errors.Field("Payment.Amount", errors.ErrMsg, "invalid amount value"))
		}
	}
	return errs
}

func (TransferPaymentChannelMsg) Path() string {
	return pathTransferPaymentChannelMsg
}

func (m *ClosePaymentChannelMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if m.ChannelID == nil {
		errs = errors.Append(errs,
			errors.Field("ChannelID", errors.ErrMsg, "missing channel ID"))
	}
	if len(m.Memo) > 128 {
		errs = errors.Append(errs,
			errors.Field("Memo", errors.ErrMsg, "memo too long"))
	}

	return errs
}

func (ClosePaymentChannelMsg) Path() string {
	return pathClosePaymentChannelMsg
}

// inThePast represents time value for Monday, January 1, 2018 2:00:00 AM GMT+01:00
//
// Assumption of this extension is that year 2018 is always in the past and it
// is safe to use as a broad border between the past and the future. This does
// not have to be a precise value as it should be used only for initial
// validation. Proper time validation must be done once the exact current time
// is available.
var inThePast weave.UnixTime = 1514768400
