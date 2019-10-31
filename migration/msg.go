package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

func init() {
	MustRegister(1, &UpgradeSchemaMsg{}, NoModification)
}

var _ weave.Msg = (*UpgradeSchemaMsg)(nil)

func (msg *UpgradeSchemaMsg) Validate() error {
	if msg.Pkg == "" {
		return errors.Wrap(errors.ErrEmpty, "pkg is required")
	}
	if msg.ToVersion == 0 {
		return errors.Wrap(errors.ErrEmpty, "to version is required")
	}
	return nil
}

func (UpgradeSchemaMsg) Path() string {
	return "migration/upgrade_schema"
}
