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
func (r Logging) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	start := time.Now()
	res, err := next.Check(ctx, store, tx)
	logDuration(ctx, start, res.Log, err, true)
	return res, err
}

// Deliver logs error -> error, success -> info
func (r Logging) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	start := time.Now()
	res, err := next.Deliver(ctx, store, tx)
	logDuration(ctx, start, res.Log, err, false)
	return res, err
}

// logDuration writes information about the time and result
// to the logger
func logDuration(ctx weave.Context, start time.Time, msg string,
	err error, lowPrio bool) {

	delta := time.Now().Sub(start)
	logger := weave.GetLogger(ctx).With("duration", micros(delta))
	if err != nil {
		logger = logger.With("err", err)
	}

	// now, write it
	if err == nil && lowPrio {
		logger.Debug(msg)
	} else if err != nil && !lowPrio {
		logger.Error(msg)
	} else { // low prio error, or normal log message
		logger.Info(msg)
	}
}

// micros returns how many microseconds passed in a call
func micros(d time.Duration) int {
	return int(d.Seconds() * 1000000)
}
