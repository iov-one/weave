package qualityscore

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (UpdateConfigurationMsg) Path() string {
	return "qualityscore/update_configuration"
}

func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Patch", m.Patch.Validate())
	return errs
}
