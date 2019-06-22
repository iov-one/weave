package distribution

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateRevenueMsg{}, migration.NoModification)
	migration.MustRegister(1, &DistributeMsg{}, migration.NoModification)
	migration.MustRegister(1, &ResetRevenueMsg{}, migration.NoModification)
}

const (
	pathCreateRevenueMsg = "distribution/createRevenue"
	pathDistributeMsg    = "distribution/distribute"
	pathResetRevenueMsg  = "distribution/resetRevenue"
)

func (msg *CreateRevenueMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if err := msg.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin address")
	}
	if err := validateRecipients(msg.Recipients, errors.ErrMsg); err != nil {
		return err
	}
	return nil
}

func (CreateRevenueMsg) Path() string {
	return pathCreateRevenueMsg
}

func (msg *DistributeMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if len(msg.RevenueID) == 0 {
		return errors.Wrap(errors.ErrMsg, "revenue ID missing")
	}
	return nil
}

func (DistributeMsg) Path() string {
	return pathDistributeMsg
}

func (msg *ResetRevenueMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if err := validateRecipients(msg.Recipients, errors.ErrMsg); err != nil {
		return err
	}
	return nil
}

func (ResetRevenueMsg) Path() string {
	return pathResetRevenueMsg
}
