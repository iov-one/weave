package distribution

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &DistributeMsg{}, migration.NoModification)
	migration.MustRegister(1, &ResetMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateMsg)(nil)

func (msg *CreateMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Admin", msg.Admin.Validate())
	errs = errors.AppendField(errs, "Destinatinos", validateDestinations(msg.Destinations, errors.ErrMsg))

	return errs
}

func (CreateMsg) Path() string {
	return "distribution/create"
}

var _ weave.Msg = (*DistributeMsg)(nil)

func (msg *DistributeMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if len(msg.RevenueID) == 0 {
		errs = errors.Append(errs, errors.Field("RevenueID", errors.ErrMsg, "revenue ID is required"))
	}

	return errs
}

func (DistributeMsg) Path() string {
	return "distribution/distribute"
}

var _ weave.Msg = (*ResetMsg)(nil)

func (msg *ResetMsg) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Destinatinos", validateDestinations(msg.Destinations, errors.ErrMsg))

	return errs
}

func (ResetMsg) Path() string {
	return "distribution/reset"
}
