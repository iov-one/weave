package client

import (
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	CurrentHeight int64 = -1
)

// Admin encapsulates operational functions to interact with the cluster.
type admin struct {
	*BnsClient
}

func Admin(c *BnsClient) *admin {
	return &admin{BnsClient: c}
}

// GetValidators returns the validator set for the given height. Height can be `negative` to return them for the latest block.
func (c *admin) GetValidators(height int64) (*ctypes.ResultValidators, error) {
	v := &height
	if height < 0 {
		v = nil
	}
	return c.BnsClient.conn.Validators(v)
}
