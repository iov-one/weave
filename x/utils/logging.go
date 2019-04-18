package utils

import (
	"time"

	"github.com/iov-one/weave"
)

// Logging is a decorator to log messages as they pass through
type Logging struct{}

var _ weave.Decorator = Logging{}

// NewLogging creates a Logging decorator
func NewLogging() Logging {
	return Logging{}
}

// Check logs error -> info, success -> debug
func (r Logging) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	start := time.Now()
	res, err := next.Check(ctx, store, tx)
	var resLog string
	if err == nil {
		resLog = res.Log
	}
	logDuration(ctx, start, resLog, err, true)
	return res, err
}

// Deliver logs error -> error, success -> info
func (r Logging) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	start := time.Now()
	res, err := next.Deliver(ctx, store, tx)
	var resLog string
	if err == nil {
		resLog = res.Log
	}
	logDuration(ctx, start, resLog, err, false)
	return res, err
}

// logDuration writes information about the time and result to the logger
func logDuration(ctx weave.Context, start time.Time, msg string, err error, lowPrio bool) {
	delta := time.Now().Sub(start)
	logger := weave.GetLogger(ctx).With("duration", delta/time.Microsecond)

	if err != nil {
		logger = logger.With("err", err)
	}

	// Although message can be empty, we still want to emit a log entry
	// because it contains other relevant information beside the message.

	if err != nil {
		logger.Error(msg)
	} else {
		if lowPrio {
			logger.Debug(msg)
		} else {
			logger.Info(msg)
		}
	}
}
