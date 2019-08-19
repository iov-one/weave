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
			Raw:        "*iov",
			WantName:   "",
			WantDomain: "iov",
		},
		"longest valid name and domain": {
			Raw:        strings.Repeat("x", 64) + "*iov",
			WantName:   strings.Repeat("x", 64),
			WantDomain: "iov",
		},
		"too long name": {
			Raw:        strings.Repeat("x", 65) + "*iov",
			WantName:   strings.Repeat("x", 65),
			WantDomain: "iov",
			WantErr:    errors.ErrInput,
		},
		"all valid characters in name": {
			Raw:        `abcdefghijklmnopqrstuvwxyz 0123456789.-_*iov`,
			WantName:   `abcdefghijklmnopqrstuvwxyz 0123456789.-_`,
			WantDomain: "iov",
			WantErr:    errors.ErrInput,
		},
		"missing domain": {
			Raw:        "foo*",
			WantErr:    errors.ErrInput,
			WantName:   "foo",
			WantDomain: "",
		},
		"missing separator": {
			Raw:        "xyz",
			WantErr:    errors.ErrInput,
			WantName:   "",
			WantDomain: "",
		},
		"invalid characters (emoji)": {
			Raw:        "😈*iov",
			WantErr:    errors.ErrInput,
			WantName:   "😈",
			WantDomain: "iov",
		},
		"invalid domain name (only iov is allowed)": {
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
