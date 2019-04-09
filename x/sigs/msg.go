package sigs

import "github.com/iov-one/weave/errors"

const (
	pathBumpSequenceMsg = "sigs/bumpSequence"

	maxSequenceIncrement = 1000
	minSequenceIncrement = 1
)

func (msg *BumpSequenceMsg) Validate() error {
	if msg.Pubkey == nil {
		return errors.Wrap(errors.ErrInvalidMsg, "public key is required")
	}
	if msg.Increment < minSequenceIncrement {
		return errors.Wrapf(errors.ErrInvalidMsg, "increment must be at least %d", minSequenceIncrement)
	}
	if msg.Increment > maxSequenceIncrement {
		return errors.Wrapf(errors.ErrInvalidMsg, "increment must not be greater than %d", maxSequenceIncrement)
	}
	return nil
}

func (BumpSequenceMsg) Path() string {
	return pathBumpSequenceMsg
}
