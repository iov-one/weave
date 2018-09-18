package username

import (
	"fmt"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIssueTokenMsgValidate(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()

	specs := []struct {
		msg      IssueTokenMsg
		expError bool
	}{
		{ // happy path email
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      []byte("alice@example.com"),
				Details: TokenDetails{},
			},
			expError: false,
		},
		{ // happy path twitter
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      []byte("@iov_official"),
				Details: TokenDetails{},
			},
			expError: false,
		},
		{ // happy path phone
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      []byte("+491234567890"),
				Details: TokenDetails{},
			},
			expError: false,
		},
		{ // owner missing
			msg: IssueTokenMsg{
				Id:      []byte("alice@example.com"),
				Details: TokenDetails{},
			},
			expError: true,
		},
		{ // owner wrong format
			msg: IssueTokenMsg{
				Owner:   []byte("not an address"),
				Id:      []byte("alice@example.com"),
				Details: TokenDetails{},
			},
			expError: true,
		},
		{ // id too short
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      []byte("foo"),
				Details: TokenDetails{},
			},
			expError: true,
		},
		{ // id too long
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      anyIDWithLength(257),
				Details: TokenDetails{},
			},
			expError: true,
		},
		{ // id with forbidden character *
			msg: IssueTokenMsg{
				Owner:   alice.Address(),
				Id:      []byte("foo*bar"),
				Details: TokenDetails{},
			},
			expError: true,
		},
		// TODO: Add checks for approvals
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.msg.Validate()
			if spec.expError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func anyIDWithLength(n int) []byte {
	r := make([]byte, n)
	for i := 0; i < n; i++ {
		r[i] = byte('a')
	}
	return r
}
