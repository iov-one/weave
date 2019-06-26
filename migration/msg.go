package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*UpgradeSchemaMsg)(nil)

func (msg *UpgradeSchemaMsg) Validate() error {
	if msg.Pkg == "" {
		return errors.Wrap(errors.ErrEmpty, "pkg is required")
	}
	return nil
}

func (UpgradeSchemaMsg) Path() string {
	return "migration/upgrade_schema"
}
