package dummy

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateCartonBoxMsg{}, migration.NoModification)
	migration.MustRegister(2, &CreateCartonBoxMsg{}, func(db weave.ReadOnlyKVStore, m migration.Migratable) error {
		c, ok := m.(*CreateCartonBoxMsg)
		if !ok {
			return errors.Wrapf(errors.ErrInvalidType, "%T", m)
		}
		c.Quality = defaultCartonBoxQuality
		return nil
	})

	migration.MustRegister(1, &InspectCartonBoxMsg{}, migration.NoModification)
	migration.MustRegister(2, &InspectCartonBoxMsg{}, migration.NoModification)
}

func (CreateCartonBoxMsg) Path() string {
	return "dummy/create_carton_box"
}

func (m *CreateCartonBoxMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if m.Width < 1 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}
	if m.Height < 1 {
		return errors.Wrap(errors.ErrInvalidMsg, "width must be greater than zero")
	}

	if m.Metadata.Schema == 1 {
		return nil
	}

	if m.Quality < 1 {
		return errors.Wrap(errors.ErrInvalidMsg, "quality must be greater than zero")
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
