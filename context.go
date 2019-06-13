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
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

type contextKey int // local to the weave module

const (
	contextKeyHeader contextKey = iota
	contextKeyHeight
	contextKeyChainID
	contextKeyLogger
	contextKeyTime
	contextCommitInfo
)

var (
	// DefaultLogger is used for all context that have not
	// set anything themselves
	DefaultLogger = log.NewNopLogger()

	// IsValidChainID is the RegExp to ensure valid chain IDs
	IsValidChainID = regexp.MustCompile(`^[a-zA-Z0-9_\-]{6,20}$`).MatchString
)

// Context is just an alias for the standard implementation.
// We use functions to extend it to our domain
type Context = context.Context

// WithHeader sets the block header for the Context.
// panics if called with header already set
func WithHeader(ctx Context, header abci.Header) Context {
	if _, ok := GetHeader(ctx); ok {
		panic("Header already set")
	}
	return context.WithValue(ctx, contextKeyHeader, header)
}

// GetHeader returns the current block header
// ok is false if no header set in this Context
func GetHeader(ctx Context) (abci.Header, bool) {
	val, ok := ctx.Value(contextKeyHeader).(abci.Header)
	return val, ok
}

// WithCommitInfo sets the info on who signed the block in this Context.
// Panics if already set.
func WithCommitInfo(ctx Context, info CommitInfo) Context {
	if _, ok := GetCommitInfo(ctx); ok {
		panic("CommitInfo already set")
	}
	return context.WithValue(ctx, contextCommitInfo, info)
}

// GetCommitInfo returns the info on validators that signed
// this block. Returns false if not present.
func GetCommitInfo(ctx Context) (CommitInfo, bool) {
	val, ok := ctx.Value(contextCommitInfo).(CommitInfo)
	return val, ok
}

// WithHeight sets the block height for the Context.
// panics if called with height already set
func WithHeight(ctx Context, height int64) Context {
	if _, ok := GetHeight(ctx); ok {
		panic("Height already set")
	}
	return context.WithValue(ctx, contextKeyHeight, height)
}

// GetHeight returns the current block height
// ok is false if no height set in this Context
func GetHeight(ctx Context) (int64, bool) {
	val, ok := ctx.Value(contextKeyHeight).(int64)
	return val, ok
}

// WithBlockTime sets the block time for the context. Block time is always
// represented in UTC.
func WithBlockTime(ctx Context, t time.Time) Context {
	return context.WithValue(ctx, contextKeyTime, t.UTC())
}

// BlockTime returns current block wall clock time as declared in the context.
// An error is returned if a block time is not present in the context or if the
// zero time value is found.
func BlockTime(ctx Context) (time.Time, error) {
	val, ok := ctx.Value(contextKeyTime).(time.Time)
	if !ok {
		return time.Time{}, errors.Wrap(errors.ErrHuman, "block time not present in the context")
	}
	if val.IsZero() {
		// This is a special case when a zero time value was attached
		// to the context. Even though it is present it is not a valid
		// value.
		return val, errors.Wrap(errors.ErrHuman, "zero value block time in the context")
	}
	return val, nil
}

// WithChainID sets the chain id for the Context.
// panics if called with chain id already set
func WithChainID(ctx Context, chainID string) Context {
	if ctx.Value(contextKeyChainID) != nil {
		panic("Chain ID already set")
	}
	if !IsValidChainID(chainID) {
		panic(fmt.Sprintf("Invalid chain ID: %s", chainID))
	}
	return context.WithValue(ctx, contextKeyChainID, chainID)
}

// GetChainID returns the current chain id
// panics if chain id not already set (should never happen)
func GetChainID(ctx Context) string {
	if x := ctx.Value(contextKeyChainID); x == nil {
		panic("Chain id is not in context")
	}
	return ctx.Value(contextKeyChainID).(string)
}

// WithLogger sets the logger for this Context
func WithLogger(ctx Context, logger log.Logger) Context {
	// Logger can be overridden below... no problem
	return context.WithValue(ctx, contextKeyLogger, logger)
}

// GetLogger returns the currently set logger, or
// DefaultLogger if none was set
func GetLogger(ctx Context) log.Logger {
	val, ok := ctx.Value(contextKeyLogger).(log.Logger)
	if !ok {
		return DefaultLogger
	}
	return val
}

// WithLogInfo accepts keyvalue pairs, and returns another
// context like this, after passing all the keyvals to the
// Logger
func WithLogInfo(ctx Context, keyvals ...interface{}) Context {
	logger := GetLogger(ctx).With(keyvals...)
	return WithLogger(ctx, logger)
}
