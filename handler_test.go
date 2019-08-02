package weave

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestOptionsStream(t *testing.T) {
	cases := map[string]struct {
		json    string
		wantErr error
		exp     []struct{ Key int }
		empty   bool
	}{
		"happy path": {
			json: `{"list": [{"key": 1}, {"key": 2}]}`,
			exp: []struct{ Key int }{
				{Key: 1},
				{Key: 2},
			},
			wantErr: errors.ErrEmpty,
		},

		"empty list": {
			json:    `{}`,
			wantErr: errors.ErrEmpty,
			empty:   true,
		},

		"wrong value": {
			json: `{"list": [{"key": "dasdasas"}]}`,
			exp: []struct{ Key int }{
				{},
			},
			wantErr: errors.ErrInput,
		},

		"wrong body": {
			json:    `{"list": "adasda"}`,
			wantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var o Options
			var s struct{ Key int }
			assert.Nil(t, json.Unmarshal([]byte(tc.json), &o))
			f, err := o.Stream("list")

			if tc.empty {
				assert.IsErr(t, tc.wantErr, err)
				return
			} else {
				assert.Nil(t, err)
			}

			for _, e := range tc.exp {
				err = f(&s)
				if err != nil {
					assert.IsErr(t, tc.wantErr, err)
					return
				}
				assert.Equal(t, e, s)
			}

			assert.IsErr(t, tc.wantErr, f(&s))
			assert.IsErr(t, errors.ErrState, f(&s))
		})
	}
}
