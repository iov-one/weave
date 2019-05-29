package weave

import (
	"context"
	"encoding/json"
	"fmt"
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
// convenient to use a string format in configurations (ie genesis file).
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
	return t.Time().UTC().String()
}

// IsExpired returns true if given time is in the past as compared to the "now"
// as declared for the block. Expiration is inclusive, meaning that if current
// time is equal to the expiration time than this function returns true.
//
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func IsExpired(ctx Context, t UnixTime) bool {
	blockNow, err := BlockTime(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return t <= AsUnixTime(blockNow)
}

// InThePast returns true if given time is in the past compared to the current
// time as declared in the context. Context "now" should come from the block
// header.
// Keep in mind that this function is not inclusive of current time. It given
// time is equal to "now" then this function returns false.
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func InThePast(ctx context.Context, t time.Time) bool {
	now, err := BlockTime(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return t.Before(now)
}

// InTheFuture returns true if given time is in the future compared to the
// current time as declared in the context. Context "now" should come from the
// block header.
// Keep in mind that this function is not inclusive of current time. It given
// time is equal to "now" then this function returns false.
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func InTheFuture(ctx context.Context, t time.Time) bool {
	now, err := BlockTime(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return t.After(now)
}

// UnixDuration represents a time duration with granularity of a second. This
// type should be used mostly for protobuf message declarations.
type UnixDuration int32

// AsUnixDuration converts given Duration into UnixDuration. Because of the
// UnixDuration granularity precision of the value is narrowed to seconds.
func AsUnixDuration(d time.Duration) UnixDuration {
	return UnixDuration(d / time.Second)
}

// Duration returns the time.Duration representation of this value.
func (d UnixDuration) Duration() time.Duration {
	return time.Duration(d) * time.Second
}

// UnmarshalJSON loads JSON serialized representation into this value. JSON
// serialized value can be represented as both number of seconds and a human
// readable string with time unit as used by the time package.
func (d *UnixDuration) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid duration string: %s", err)
		}
		*d = AsUnixDuration(dur)
		return nil
	}

	var n int32
	if err := json.Unmarshal(raw, &n); err != nil {
		return err
	}
	*d = UnixDuration(n)
	return nil
}

func (d UnixDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(d))
}

func (d UnixDuration) String() string {
	return d.Duration().String()
}
