package weave

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/iov-one/weave/errors"
)

func TestUnixTimeUnmarshal(t *testing.T) {
	cases := map[string]struct {
		raw      string
		wantTime UnixTime
		wantErr  *errors.Error
	}{
		"zero time as number": {
			raw:      "0",
			wantTime: 0,
		},
		"zero time as string": {
			raw:      `"1970-01-01T01:00:00+01:00"`,
			wantTime: 0,
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
			raw:     "-1",
			wantErr: errors.ErrInvalidInput,
		},
		"negative time as string": {
			raw:     `"1950-01-01T01:00:00+01:00"`,
			wantErr: errors.ErrInvalidInput,
		},
		"invalid string": {
			raw:     `"not a time string"`,
			wantErr: errors.ErrInvalidInput,
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
	now := time.Now()
	future := now.Add(time.Hour + 4*time.Second)

	unow := AsUnixTime(now)
	ufuture := unow.Add(time.Hour + 4*time.Second)

	if future.Unix() != int64(ufuture) {
		t.Fatalf("want %d, got %d", future.Unix(), ufuture)
	}
}
