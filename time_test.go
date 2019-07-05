package weave

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUnixTimeUnmarshal(t *testing.T) {
	cases := map[string]struct {
		raw      string
		wantTime UnixTime
		wantErr  *errors.Error
	}{
		"zero UNIX time as number": {
			raw:      "0",
			wantTime: 0,
		},
		"zero UNIX time as string": {
			raw:      `"1970-01-01T01:00:00+01:00"`,
			wantTime: 0,
		},
		"zero time as number": {
			raw:      "-62135596800",
			wantTime: -62135596800,
		},
		"zero time as string": {
			raw:      `"0001-01-01T00:00:00Z"`,
			wantTime: -62135596800,
		},
		"a time as string": {
			raw:      `"2019-04-04T11:35:40.89181085+02:00"`,
			wantTime: 1554370540,
		},
		"a time as number": {
			raw:      "1554370540",
			wantTime: 1554370540,
		},
		"negative number": {
			raw:      "-1",
			wantTime: -1,
		},
		"negative time as string": {
			raw:      `"1970-01-01T00:59:59+01:00"`,
			wantTime: -1,
		},
		"invalid string": {
			raw:     `"not a time string"`,
			wantErr: errors.ErrInput,
		},
		"string as futuristic as it gets": {
			raw:      `"9999-12-31T23:59:59Z"`,
			wantTime: maxUnixTime,
		},
		"number as futuristic as it gets": {
			raw:      "253402300799",
			wantTime: maxUnixTime,
		},
		"string too much in the future": {
			raw:     `"10000-01-01T01:01:01Z"`,
			wantErr: errors.ErrInput,
		},
		"number too much in the future": {
			raw:     "253402300800",
			wantErr: errors.ErrState,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got UnixTime
			err := json.Unmarshal([]byte(tc.raw), &got)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != tc.wantTime {
				t.Fatalf("want %d time, got %d", tc.wantTime, got)
			}
		})
	}
}

func TestUnixTimeAdd(t *testing.T) {
	cases := map[string]struct {
		base  UnixTime
		delta time.Duration
		want  UnixTime
	}{
		"zero delta": {
			base:  123,
			delta: 0,
			want:  123,
		},
		"add less than a second must not modify the value": {
			base:  123,
			delta: 999 * time.Millisecond,
			want:  123,
		},
		"subtract less than a second must not modify the value": {
			base:  123,
			delta: -999 * time.Millisecond,
			want:  123,
		},
		"add more than a second must add only full seconds": {
			base:  123,
			delta: 2999 * time.Millisecond,
			want:  125,
		},
		"subtract more than a second must subtract only full seconds": {
			base:  123,
			delta: -2999 * time.Millisecond,
			want:  121,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := tc.base.Add(tc.delta)
			if got != tc.want {
				t.Fatalf("unexpected result: %d", got)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	now := AsUnixTime(time.Now())
	ctx := WithBlockTime(context.Background(), now.Time())

	future := now.Add(5 * time.Minute)
	if IsExpired(ctx, future) {
		t.Error("future is expired")
	}

	past := now.Add(-5 * time.Minute)
	if !IsExpired(ctx, past) {
		t.Error("past is not expired")
	}

	if !IsExpired(ctx, now) {
		t.Fatal("when expiration time is equal to now it is expected to be expired")
	}
}

func TestIsExpiredRequiresBlockTime(t *testing.T) {
	now := AsUnixTime(time.Now())
	assert.Panics(t, func() {
		// Calling isExpected with a context without a block height
		// attached is expected to panic.
		IsExpired(context.Background(), now)
	})
}

func TestUnixDurationJSONUnmarshal(t *testing.T) {
	cases := map[string]struct {
		raw     string
		wantDur UnixDuration
		wantErr *errors.Error
	}{
		"zero string ": {
			raw:     `"0s"`,
			wantDur: 0,
		},
		"zero number ": {
			raw:     `0`,
			wantDur: 0,
		},
		"string notation": {
			raw:     `"2h"`,
			wantDur: AsUnixDuration(2 * time.Hour),
		},
		"negative string notation": {
			raw:     `"-2m"`,
			wantDur: AsUnixDuration(-2 * time.Minute),
		},
		"number notation": {
			raw:     `2`,
			wantDur: AsUnixDuration(2 * time.Second),
		},
		"negative number notation": {
			raw:     `-123`,
			wantDur: AsUnixDuration(-123 * time.Second),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got UnixDuration
			err := json.Unmarshal([]byte(tc.raw), &got)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != tc.wantDur {
				t.Fatalf("want %s duration, got %s", tc.wantDur, got)
			}
		})
	}
}

func TestInThePast(t *testing.T) {
	now := time.Now()
	ctx := WithBlockTime(context.Background(), now)

	assert.Equal(t, false, InThePast(ctx, now))
	assert.Equal(t, false, InThePast(ctx, now.Add(time.Second)))
	assert.Equal(t, true, InThePast(ctx, now.Add(-time.Second)))
}

func TestInThePastRequiresBlockTime(t *testing.T) {
	assert.Panics(t, func() {
		// Calling isExpected with a context without a block height
		// attached is expected to panic.
		InThePast(context.Background(), time.Now())
	})
}

func TestInTheFuture(t *testing.T) {
	now := time.Now()
	ctx := WithBlockTime(context.Background(), now)

	assert.Equal(t, false, InTheFuture(ctx, now))
	assert.Equal(t, true, InTheFuture(ctx, now.Add(time.Second)))
	assert.Equal(t, false, InTheFuture(ctx, now.Add(-time.Second)))
}

func TestInTheFutureRequiresBlockTime(t *testing.T) {
	assert.Panics(t, func() {
		// Calling isExpected with a context without a block height
		// attached is expected to panic.
		InTheFuture(context.Background(), time.Now())
	})
}
