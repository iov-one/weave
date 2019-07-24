package username

import (
	"strings"
	"testing"

	"github.com/iov-one/weave/errors"
)

func TestUsername(t *testing.T) {
	cases := map[string]struct {
		Raw        string
		WantErr    *errors.Error
		WantName   string
		WantDomain string
	}{
		"success": {
			Raw:        "alice*iov",
			WantName:   "alice",
			WantDomain: "iov",
		},
		"shortest valid name and domain": {
			Raw:        strings.Repeat("x", 4) + "*iov",
			WantName:   strings.Repeat("x", 4),
			WantDomain: "iov",
		},
		"longest valid name and domain": {
			Raw:        strings.Repeat("x", 64) + "*iov",
			WantName:   strings.Repeat("x", 64),
			WantDomain: "iov",
		},
		"too long name": {
			Raw:        strings.Repeat("x", 65) + "*" + strings.Repeat("x", 6),
			WantName:   strings.Repeat("x", 65),
			WantDomain: strings.Repeat("x", 6),
			WantErr:    errors.ErrInput,
		},
		"too long domain": {
			Raw:        strings.Repeat("x", 8) + "*" + strings.Repeat("x", 17),
			WantName:   strings.Repeat("x", 8),
			WantDomain: strings.Repeat("x", 17),
			WantErr:    errors.ErrInput,
		},
		"too short name": {
			Raw:        strings.Repeat("x", 3) + "*" + strings.Repeat("x", 6),
			WantName:   strings.Repeat("x", 3),
			WantDomain: strings.Repeat("x", 6),
			WantErr:    errors.ErrInput,
		},
		"too short domain": {
			Raw:        strings.Repeat("x", 8) + "*" + strings.Repeat("x", 2),
			WantName:   strings.Repeat("x", 8),
			WantDomain: strings.Repeat("x", 2),
			WantErr:    errors.ErrInput,
		},
		"missing domain": {
			Raw:        "foo*",
			WantErr:    errors.ErrInput,
			WantName:   "foo",
			WantDomain: "",
		},
		"missing name": {
			Raw:        "*iov",
			WantErr:    errors.ErrInput,
			WantName:   "",
			WantDomain: "iov",
		},
		"missing separator": {
			Raw:        "xyz",
			WantErr:    errors.ErrInput,
			WantName:   "",
			WantDomain: "",
		},
		"invalid characters (emoji)": {
			Raw:        "😈*😀",
			WantErr:    errors.ErrInput,
			WantName:   "😈",
			WantDomain: "😀",
		},
		"invalid domain name": {
			Raw:        "extreme*expert",
			WantErr:    errors.ErrInput,
			WantName:   "extreme",
			WantDomain: "expert",
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			u, err := ParseUsername(tc.Raw)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %v", err)
			}

			if n := u.Name(); n != tc.WantName {
				t.Fatalf("unexpected name: %q", n)
			}
			if d := u.Domain(); d != tc.WantDomain {
				t.Fatalf("unexpected domain: %q", d)
			}
		})
	}
}
