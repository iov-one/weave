package validators

import (
	"github.com/iov-one/weave/errors"
)

// Error codes
// x/update_validators reserves 140 ~ 149.

var (
	ErrEmptyDiff     = errors.Register(140, "empty validator diff")
	ErrInvalidPubKey = errors.Register(141, "invalid public key")
	ErrInvalidPower  = errors.Register(142, "power value is invalid")
)
