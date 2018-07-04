package validators

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/orm"
	abci "github.com/tendermint/abci/types"
)

// Controller is the functionality needed by
// cash.Handler and cash.Decorator. BaseController
// should work plenty fine, but you can add other logic
// if so desired
type Controller interface {
	CanUpdateValidators(store weave.KVStore, checkAddress CheckAddress, diff []abci.Validator) ([]abci.Validator, error)
}

// BaseController is a simple implementation of controller
// wallet must return something that supports AsSet
type BaseController struct {
	bucket orm.Bucket
}

// NewController returns a basic controller implementation
func NewController(bucket orm.Bucket) BaseController {
	return BaseController{bucket: bucket}
}

func (c BaseController) CanUpdateValidators(store weave.KVStore, checkAddress CheckAddress, diff []abci.Validator) ([]abci.Validator, error) {
	if len(diff) == 0 {
		return nil, ErrEmptyDiff()
	}

	accts, err := GetAccounts(c.bucket, store)
	if err != nil {
		return nil, err
	}

	ok, err := HasPermission(accts, checkAddress)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, errors.ErrUnauthorized()
	}

	return diff, nil
}
