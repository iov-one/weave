package weave

import (
	"encoding/json"
	"time"

	"github.com/iov-one/weave/errors"
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

// IsZero returns true if this time represents a zero value.
func (t UnixTime) IsZero() bool {
	return t == 0
}

// Add modifies this UNIX time by given duration. This is compatible with
// time.Time.Add method.
func (t UnixTime) Add(d time.Duration) UnixTime {
	return t + UnixTime(d/time.Second)
}

// AsUnixTime converts given Time structure into its UNIX time representation.
func AsUnixTime(t time.Time) UnixTime {
	return UnixTime(t.Unix())
}

// UnmarshalJSON supports unmarshaling both as time.Time and from a number.
// Usually a number is used as a representation of this time in JSON but it is
// convinient to use a string format in configurations (ie genesis file).
func (t *UnixTime) UnmarshalJSON(raw []byte) error {
	var unix int64
	if err := json.Unmarshal(raw, &unix); err == nil {
		if unix < 0 {
			return errors.Wrap(errors.ErrInvalidInput, "time before epoch")
		}
		*t = UnixTime(unix)
		return nil
	}

	var stdtime time.Time
	if err := json.Unmarshal(raw, &stdtime); err == nil {
		unix := UnixTime(stdtime.Unix())
		if unix < 0 {
			return errors.Wrap(errors.ErrInvalidInput, "time before epoch")
		}
		*t = unix
		return nil
	}

	return errors.Wrap(errors.ErrInvalidInput, "invalid time format")
}

// Validate returns an error if this time value is invalid.
func (t UnixTime) Validate() error {
	if t < 0 {
		return errors.Wrap(errors.ErrInvalidState, "negative value")
	}
	return nil
}

// String returns the usual string representation of this time as the time.Time
// structure would.
func (t UnixTime) String() string {
	return t.Time().String()
}
