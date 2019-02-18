package orm

import (
	"github.com/iov-one/weave/errors"
)

// Orm reserves 100~109 error codes

// InvalidIndexErr is returned when an index specified is invalid
var InvalidIndexErr = errors.Register(100, "invalid index")

