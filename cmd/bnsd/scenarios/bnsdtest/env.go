package bnsdtest

import (
	"io"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
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

	msgfees    map[string]coin.Coin
	governance governance
	usernames  map[string]username.Token

	Client         client.Client
	clientThrottle time.Duration

	MultiSigContract  weave.Condition
	EscrowContract    weave.Condition
	DistrContractAddr weave.Address
	Node              *nm.Node
	Logger            log.Logger
	RpcAddress        string
}

// IsRemote returns true if we connect to a remote chain (not local CI test), in order to skip some tests
func (e *EnvConf) IsRemote() bool {
	return e.Node == nil
}

type governance struct {
	electors     []weave.Address
	votingPeriod weave.UnixDuration
}

func WithMinFee(c coin.Coin) StartBnsdOption {
	return func(env *EnvConf) {
		env.MinFee = c
	}
}

func WithAntiSpamFee(c coin.Coin) StartBnsdOption {
	return func(env *EnvConf) {
		env.AntiSpamFee = c
	}
}

func WithLogger(out io.Writer) StartBnsdOption {
	return func(env *EnvConf) {
		env.Logger = log.NewTMLogger(out)
	}
}

func WithThrottle(frequency time.Duration) StartBnsdOption {
	return func(env *EnvConf) {
		env.clientThrottle = frequency
	}
}

func WithMsgFee(msgPath string, fee coin.Coin) StartBnsdOption {
	return func(env *EnvConf) {
		env.msgfees[msgPath] = fee
	}
}

func WithUsername(name string, token username.Token) StartBnsdOption {
	return func(env *EnvConf) {
		env.usernames[name] = token
	}
}

// WithGovernance sets given group of weave addresses as the electorate for the
// first electorate instance created. First address is used as the admin for
// the electorate and the governance rule.
func WithGovernance(votingPeriod weave.UnixDuration, electors []weave.Address) StartBnsdOption {
	return func(env *EnvConf) {
		env.governance.votingPeriod = votingPeriod
		env.governance.electors = electors
	}
}
