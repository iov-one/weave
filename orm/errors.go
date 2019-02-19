package orm

import (
	"github.com/iov-one/weave/errors"
)

// Orm reserves 100~109 error codes

// ErrInvalidIndex is returned when an index specified is invalid
var ErrInvalidIndex = errors.Register(100, "invalid index")
