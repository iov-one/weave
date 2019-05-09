package weave

import (
	"encoding/json"
	"time"

	"github.com/iov-one/weave/errors"
)

const (
	// UNIX time value of 0001-01-01T00:00:00Z
	minUnixTime = -62135596800

	// UNIX time value of 9999-12-31T23:59:59Z
	maxUnixTime = 253402300799
)

// UnixTime represents a point in time as POSIX time.
// This type comes in handy when dealing with protobuf messages. Instead of
// using Go's time.Time that includes nanoseconds use primitive int64 type and
// seconds precision. Some languages do not support nanoseconds precision
// anyway.
//
// When using in protobuf declaration, use gogoproto's typecasting
//
//   int64 deadline = 1 [(gogoproto.casttype) = "github.com/iov-one/weave.UnixTime"];
//
type UnixTime int64

// Time returns a time.Time structure that represents the same moment in time.
func (t UnixTime) Time() time.Time {
	return time.Unix(int64(t), 0)
}

// Add modifies this UNIX time by given duration. This is compatible with
// time.Time.Add method. Any duration value smaller than a second is ignored as
// it cannot be represented by the UnixTime type.
func (t UnixTime) Add(d time.Duration) UnixTime {
	return t + UnixTime(d/time.Second)
}

// AsUnixTime converts given Time structure into its UNIX time representation.
// All time information more granular than a second is dropped as it cannot be
// represented by the UnixTime type.
func AsUnixTime(t time.Time) UnixTime {
	return UnixTime(t.Unix())
}

// UnmarshalJSON supports unmarshaling both as time.Time and from a number.
// Usually a number is used as a representation of this time in JSON but it is
// convinient to use a string format in configurations (ie genesis file).
// Any granularity smaller than a second is dropped. For example, 1900
// milliseconds will be narrowed to 1 second.
func (t *UnixTime) UnmarshalJSON(raw []byte) error {
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		unix := UnixTime(n)
		if err := unix.Validate(); err != nil {
			return err
		}
		*t = unix
		return nil
	}

	var stdtime time.Time
	if err := json.Unmarshal(raw, &stdtime); err == nil {
		unix := UnixTime(stdtime.Unix())
		if err := unix.Validate(); err != nil {
			return err
		}
		*t = unix
		return nil
	}

	return errors.Wrap(errors.ErrInput, "invalid time format")
}

// Validate returns an error if this time value is invalid.
func (t UnixTime) Validate() error {
	if t < minUnixTime {
		return errors.Wrap(errors.ErrState, "time must be an A.D. value")
	}
	if t > maxUnixTime {
		return errors.Wrap(errors.ErrState, "time must be an before year 10000")
	}
	return nil
}

// String returns the usual string representation of this time as the time.Time
// structure would.
func (t UnixTime) String() string {
	return t.Time().String()
}

// IsExpired returns true if given time is in the past as compared to the "now"
// as declared for the block. Expiration is inclusive, meaning that if current
// time is equal to the expiration time than this function returns true.
//
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func IsExpired(ctx Context, t UnixTime) bool {
	blockNow, ok := BlockTime(ctx)
	if !ok {
		panic("block time is not present")
	}
	return t <= AsUnixTime(blockNow)
}
