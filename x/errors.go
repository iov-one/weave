package x

import (
	"github.com/iov-one/weave/errors"
)

// Reserved codes 130~139
var (
	ErrInvalidCurrency = errors.Register(130, "invalid currency code")
	ErrInvalidCoin     = errors.Register(131, "invalid coin")
	ErrInvalidWallet   = errors.Register(132, "invalid wallet")
)

const (
	outOfRange = "overflow coin range"
)
