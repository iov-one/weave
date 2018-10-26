package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	abci "github.com/tendermint/tendermint/abci/types"
)

type CheckAddress func(address weave.Address) bool

// Controller is the functionality needed by
// cash.Handler and cash.Decorator. BaseController
// should work plenty fine, but you can add other logic
// if so desired
type Controller interface {
	CanUpdateValidators(store weave.KVStore, checkAddress CheckAddress, diff []abci.ValidatorUpdate) ([]abci.ValidatorUpdate, error)
}

// BaseController is a simple implementation of controller
// wallet must return something that supports AsSet
type BaseController struct {
	bucket orm.Bucket
}

// NewController returns a basic controller implementation
func NewController() BaseController {
	return BaseController{bucket: NewBucket()}
}

func (c BaseController) CanUpdateValidators(store weave.KVStore, checkAddress CheckAddress, diff []abci.ValidatorUpdate) ([]abci.ValidatorUpdate, error) {
	if len(diff) == 0 {
		return nil, ErrEmptyDiff()
	}

	accts, err := GetAccounts(c.bucket, store)
	if err != nil {
		return nil, err
	}

	ok := HasPermission(AsWeaveAccounts(accts), checkAddress)
	if !ok {
		return nil, errors.ErrUnauthorized()
	}

	return diff, nil
}
