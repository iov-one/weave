package validators

import (
	"github.com/iov-one/weave/errors"
)

// Error codes
// x/update_validators reserves 140 ~ 149.

var (
	EmptyDiffErr     = errors.Register(140, "empty validator diff")
	InvalidPubKeyErr = errors.Register(141, "invalid public key")
	InvalidPower     = errors.Register(142, "power value is invalid")
)
