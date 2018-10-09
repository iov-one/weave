package ticker_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/stretchr/testify/assert"
)

func TestIssueTokenMsgValidate(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	specs := []struct {
		token    ticker.IssueTokenMsg
		expError bool
	}{
		{ // happy path
			token: ticker.IssueTokenMsg{
				Id:      []byte("BTC"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{[]byte("myBlockchainID")},
			},
			expError: false,
		},
		{ // happy path
			token: ticker.IssueTokenMsg{
				Id:      []byte("ANY1"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{[]byte("myBlockchainID")},
			},
			expError: false,
		},
		{ // not an address
			token: ticker.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   []byte("not an address"),
				Details: ticker.TokenDetails{[]byte("myBlockchainID")},
			},
			expError: true,
		},
		{ // id to small
			token: ticker.IssueTokenMsg{
				Id:      []byte("FO"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{[]byte("myBlockchainID")},
			},
			expError: true,
		},
		{ // id too big
			token: ticker.IssueTokenMsg{
				Id:      []byte("FOOBA"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{[]byte("myBlockchainID")},
			},
			expError: true,
		},
		{ // empty payload
			token: ticker.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{},
			},
			expError: true,
		},
		{ // invalid payload
			token: ticker.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   alice.Address(),
				Details: ticker.TokenDetails{[]byte("&&&")},
			},
			expError: true,
		},
	}
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := spec.token.Validate()
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
