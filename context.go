/*
Package weave defines all common interfaces to weave
together the various subpackages, as well as
implementations of some of the simpler components
(when interfaces would be too much overhead).

We pass context through context.Context between
app, middleware, and handlers. To do so, weave defines
some common keys to store info, such as block height and
chain id. Each extension, such as auth, may add its own
keys to enrich the context with specific data.

There should exist two functions for every XYZ of type T
that we want to support in Context:

  WithXYZ(Context, T) Context
  GetXYZ(Context) (val T, ok bool)

WithXYZ may error/panic if the value was previously set
to avoid lower-level modules overwriting the value
(eg. height, header)
*/
package weave

import (
	"regexp"
	"time"

	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	// DefaultLogger is used for all context that have not
	// set anything themselves
	DefaultLogger = log.NewNopLogger()

	// IsValidChainID is the RegExp to ensure valid chain IDs
	IsValidChainID = regexp.MustCompile(`^[a-zA-Z0-9_\-]{6,20}$`).MatchString
)

type BlockInfo struct {
	header     abci.Header
	commitInfo CommitInfo
	chainID    string
	logger     log.Logger
}

// NewBlockInfo creates a BlockInfo struct with current context of where it is being executed
func NewBlockInfo(header abci.Header, commitInfo CommitInfo, chainID string, logger log.Logger) (BlockInfo, error) {
	if !IsValidChainID(chainID) {
		return BlockInfo{}, errors.Wrap(errors.ErrInput, "chainID invalid")
	}
	if logger == nil {
		logger = DefaultLogger
	}
	return BlockInfo{
		header:     header,
		commitInfo: commitInfo,
		chainID:    chainID,
		logger:     logger,
	}, nil
}

func (b BlockInfo) Header() abci.Header {
	return b.header
}

func (b BlockInfo) CommitInfo() CommitInfo {
	return b.commitInfo
}

func (b BlockInfo) ChainID() string {
	return b.chainID
}

func (b BlockInfo) Height() int64 {
	return b.header.Height
}

func (b BlockInfo) BlockTime() time.Time {
	return b.header.Time
}

func (b BlockInfo) UnixTime() UnixTime {
	return AsUnixTime(b.header.Time)
}

func (b BlockInfo) Logger() log.Logger {
	return b.logger
}

// WithLogInfo accepts keyvalue pairs, and returns another
// context like this, after passing all the keyvals to the
// Logger
func (b BlockInfo) WithLogInfo(keyvals ...interface{}) BlockInfo {
	b.logger = b.logger.With(keyvals...)
	return b
}

// IsExpired returns true if given time is in the past as compared to the "now"
// as declared for the block. Expiration is inclusive, meaning that if current
// time is equal to the expiration time than this function returns true.
func (b BlockInfo) IsExpired(t UnixTime) bool {
	return t <= b.UnixTime()
}

// InThePast returns true if given time is in the past compared to the current
// time as declared in the context. Context "now" should come from the block
// header.
// Keep in mind that this function is not inclusive of current time. It given
// time is equal to "now" then this function returns false.
func (b BlockInfo) InThePast(t time.Time) bool {
	return t.Before(b.BlockTime())
}

// InTheFuture returns true if given time is in the future compared to the
// current time as declared in the context. Context "now" should come from the
// block header.
// Keep in mind that this function is not inclusive of current time. It given
// time is equal to "now" then this function returns false.
func (b BlockInfo) InTheFuture(t time.Time) bool {
	return t.After(b.BlockTime())
}
