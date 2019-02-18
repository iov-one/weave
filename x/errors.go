package x

import (
	"github.com/iov-one/weave/errors"
)

// Reserved codes 130~139
var (
	InvalidCurrencyErr = errors.Register(130, "invalid currency code")
	InvalidCoinErr     = errors.Register(131, "invalid coin")
	InvalidWalletErr   = errors.Register(132, "invalid wallet")
)

const (
	outOfRangeErr = "overflow coin range"
)
