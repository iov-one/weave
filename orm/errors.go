package orm

import (
	"github.com/iov-one/weave/errors"
)

// Orm reserves 100~109 error codes

// ErrInvalidIndex is returned when an index specified is invalid
var ErrInvalidIndex = errors.Register(100, "invalid index")

// ErrBucket is returned when already initialized bucket is tried
// to be indexed again
var ErrBucket = errors.Register(101, "bucket already initialized")
