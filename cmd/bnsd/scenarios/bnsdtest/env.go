package bnsdtest

import (
	"io"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
)

// EnvConf is a work in progress collection of previously global variables.
// This is to be cleaned up in the following updates.
type EnvConf struct {
	Alice       *client.PrivateKey
	ChainID     string
	AntiSpamFee coin.Coin
	MinFee      coin.Coin

	Client         client.Client
	clientThrottle time.Duration

	MultiSigContract  weave.Condition
	EscrowContract    weave.Condition
	DistrContractAddr weave.Address
	Node              *nm.Node
	Logger            log.Logger
	RpcAddress        string
}

func EnvMinFee(c coin.Coin) StartBnsdOption {
	return func(env *EnvConf) {
		env.MinFee = c
	}
}

func EnvLogger(out io.Writer) StartBnsdOption {
	return func(env *EnvConf) {
		env.Logger = log.NewTMLogger(out)
	}
}

func EnvThrottle(frequency time.Duration) StartBnsdOption {
	return func(env *EnvConf) {
		env.clientThrottle = frequency
	}
}
