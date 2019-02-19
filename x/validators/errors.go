package validators

import (
	"github.com/iov-one/weave/errors"
)

// Error codes
// x/update_validators reserves 140 ~ 149.

var (
	EmptyDiffErr = errors.Register(140, "empty validator diff")
	InvalidPubKeyErr = errors.Register(141, "invalid public key")
	EmptyValidatorErr = errors.Register(142, "empty validator set")
	InvalidPower = errors.Register(143, "power value is invalid")
)