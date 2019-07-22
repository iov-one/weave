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
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if msg.Increment < minSequenceIncrement {
		errs = errors.Append(errs,
			errors.Field("Increment", errors.ErrMsg, "increment must be at least %d", minSequenceIncrement))
	}
	if msg.Increment > maxSequenceIncrement {
		return errors.Append(errs,
			errors.Field("Increment", errors.ErrMsg, "increment must not be greater than %d", maxSequenceIncrement))
	}
	return errs
}

func (BumpSequenceMsg) Path() string {
	return "sigs/bump_sequence"
}
