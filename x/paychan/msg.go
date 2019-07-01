package paychan

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferMsg{}, migration.NoModification)
	migration.MustRegister(1, &CloseMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateMsg)(nil)

func (m *CreateMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Source", m.Source.Validate())
	if m.SourcePubkey == nil {
		errs = errors.Append(errs,
			errors.Field("SourcePubKey", errors.ErrMsg, "missing source public key"))
	}
	errs = errors.AppendField(errs, "Destination", m.Destination.Validate())
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

func (CreateMsg) Path() string {
	return "paychan/create"
}

var _ weave.Msg = (*TransferMsg)(nil)

func (m *TransferMsg) Validate() error {
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

func (TransferMsg) Path() string {
	return "paychan/transfer"
}

var _ weave.Msg = (*CloseMsg)(nil)

func (m *CloseMsg) Validate() error {
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

func (CloseMsg) Path() string {
	return "paychan/close"
}

// inThePast represents time value for Monday, January 1, 2018 2:00:00 AM GMT+01:00
//
// Assumption of this extension is that year 2018 is always in the past and it
// is safe to use as a broad border between the past and the future. This does
// not have to be a precise value as it should be used only for initial
// validation. Proper time validation must be done once the exact current time
// is available.
var inThePast weave.UnixTime = 1514768400
