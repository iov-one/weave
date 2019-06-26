package distribution

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &DistributeMsg{}, migration.NoModification)
	migration.MustRegister(1, &ResetMsg{}, migration.NoModification)
}

func (msg *CreateMsg) Validate() error {
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

func (CreateMsg) Path() string {
	return "distribution/create"
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
	return "distribution/distribute"
}

func (msg *ResetMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if err := validateRecipients(msg.Recipients, errors.ErrMsg); err != nil {
		return err
	}
	return nil
}

func (ResetMsg) Path() string {
	return "distribution/reset"
}
