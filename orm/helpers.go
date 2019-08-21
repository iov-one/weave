package orm

import (
	"github.com/iov-one/weave/errors"
)

// ValidateSequence returns an error if this is not an 8-byte
// as expected for orm.IDGenBucket
func ValidateSequence(id []byte) error {
	if len(id) == 0 {
		return errors.Wrap(errors.ErrEmpty, "sequence missing")
	}
	if len(id) != 8 {
		return errors.Wrap(errors.ErrInput, "sequence is invalid length (expect 8 bytes)")
	}
	return nil
}
