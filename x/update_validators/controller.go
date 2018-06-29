package update_validators

import (
	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	abci "github.com/tendermint/abci/types"
)

// Controller is the functionality needed by
// cash.Handler and cash.Decorator. BaseController
// should work plenty fine, but you can add other logic
// if so desired
type Controller interface {
	UpdateValidators(store weave.KVStore, diff []abci.Validator)
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

func (c BaseController) UpdateValidators(store weave.KVStore, user weave.Address, diff []abci.Validator) ([]abci.Validator, error) {
	if len(diff) == 0 {
		return nil, ErrEmptyDiff()
	}

	accts, err := GetAccounts(c.bucket, store)
	if err != nil {
		return nil, err
	}

	ok, err := HasPermission(accts, user)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrUnauthorized(user.String())
	}

	return diff, nil
}
