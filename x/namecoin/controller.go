package namecoin

import "github.com/iov-one/weave/x/cash"

// NewController uses the default implementation for now.
//
// TODO: better enforce token presence and sigfigs
func NewController() cash.Controller {
	return cash.NewController(NewWalletBucket())
}
