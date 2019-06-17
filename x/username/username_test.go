package username

import (
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
		"missing domain": {
			Raw:        "foo*",
			WantErr:    errors.ErrInput,
			WantName:   "foo",
			WantDomain: "",
		},
		"missing name": {
			Raw:        "*foo",
			WantErr:    errors.ErrInput,
			WantName:   "",
			WantDomain: "foo",
		},
		"missing separator": {
			Raw:        "xyz",
			WantErr:    errors.ErrInput,
			WantName:   "",
			WantDomain: "",
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
