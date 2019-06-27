package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &BumpSequenceMsg{}, migration.NoModification)
}

const (
	maxSequenceIncrement = 1000
	minSequenceIncrement = 1
)

var _ weave.Msg = (*BumpSequenceMsg)(nil)

func (msg *BumpSequenceMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if msg.Increment < minSequenceIncrement {
		return errors.Wrapf(errors.ErrMsg, "increment must be at least %d", minSequenceIncrement)
	}
	if msg.Increment > maxSequenceIncrement {
		return errors.Wrapf(errors.ErrMsg, "increment must not be greater than %d", maxSequenceIncrement)
	}
	return nil
}

func (BumpSequenceMsg) Path() string {
	return "sigs/bump_sequence"
}
