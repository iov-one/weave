package dummy

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateCartonBoxMsg{}, migration.NoModification)
	migration.MustRegister(1, &InspectCartonBoxMsg{}, migration.NoModification)
}

func (CreateCartonBoxMsg) Path() string {
	return "dummy/create_carton_box"
}

func (m *CreateCartonBoxMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if m.Width <= 0 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}
	if m.Height <= 0 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}
	return nil
}

func (InspectCartonBoxMsg) Path() string {
	return "dummy/inspect_carton_box"
}

func (m *InspectCartonBoxMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	return nil
}
