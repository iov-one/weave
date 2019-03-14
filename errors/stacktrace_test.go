package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStackTrace(t *testing.T) {
	cases := map[string]struct {
		err       error
		wantError string
	}{
		"New gives us a stacktrace": {
			err:       Wrap(ErrDuplicate, "name"),
			wantError: "name: duplicate",
		},
		"Wrapping stderr gives us a stacktrace": {
			err:       Wrap(fmt.Errorf("foo"), "standard"),
			wantError: "standard: foo",
		},
		"Wrapping pkg/errors gives us clean stacktrace": {
			err:       Wrap(errors.New("bar"), "pkg"),
			wantError: "pkg: bar",
		},
		"Wrapping inside another function is still clean": {
			err:       Wrap(fmt.Errorf("indirect"), "do the do"),
			wantError: "do the do: indirect",
		},
	}

	// Wrapping code is unwanted in the errors stack trace.
	unwantedSrc := []string{
		"github.com/iov-one/weave/errors.Wrap\n",
		"github.com/iov-one/weave/errors.Wrapf\n",
		"github.com/iov-one/weave/errors.Error.New\n",
		"github.com/iov-one/weave/errors.Error.Newf\n",
		"runtime.goexit\n",
	}
	const thisTestSrc = "github.com/iov-one/weave/errors/stacktrace_test.go"

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.err.Error(), tc.wantError)

			assert.NotNil(t, stackTrace(tc.err))

			fullStack := fmt.Sprintf("%+v", tc.err)
			if !strings.Contains(fullStack, thisTestSrc) {
				t.Logf("Stack trace below\n----%s\n----", fullStack)
				t.Error("full stack trace should contain this test source code information")
			}
			if !strings.Contains(fullStack, tc.wantError) {
				t.Logf("Stack trace below\n----%s\n----", fullStack)
				t.Error("full stack trace should contain the error description")
			}
			for _, src := range unwantedSrc {
				if strings.Contains(fullStack, src) {
					t.Logf("Stack trace below\n----%s\n----", fullStack)
					t.Logf("full stack contains unwanted source file path: %q", src)
				}
			}

			tinyStack := fmt.Sprintf("%v", tc.err)
			assert.True(t, strings.HasPrefix(tinyStack, tc.wantError))
			assert.False(t, strings.Contains(tinyStack, "\n"), "only one line is expected")
			// contains a link to where it was created, which must
			// be here, not the Wrap() function
			assert.True(t, strings.Contains(tinyStack, "[iov-one/weave/errors/stacktrace_test.go"))
		})
	}
}
