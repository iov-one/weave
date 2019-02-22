package sigs

import (
	"github.com/iov-one/weave/errors"
)

var (
	ErrInvalidSequence = errors.Register(120, "invalid sequence number")
)
