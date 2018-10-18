package bootstrap_node_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft/bootstrap_node"
	"github.com/stretchr/testify/assert"
)

func TestIssueTokenMsgValidate(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	specs := []struct {
		token    bootstrap_node.IssueTokenMsg
		expError bool
	}{
		{ // happy path
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("BTC"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{[]byte("myBlockchainID"), bootstrap_node.URI{}},
			},
			expError: false,
		},
		{ // happy path
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("ANY1"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{[]byte("myBlockchainID"), bootstrap_node.URI{}},
			},
			expError: false,
		},
		{ // not an address
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   []byte("not an address"),
				Details: bootstrap_node.TokenDetails{[]byte("myBlockchainID"), bootstrap_node.URI{}},
			},
			expError: true,
		},
		{ // id to small
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("FO"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{[]byte("myBlockchainID"), bootstrap_node.URI{}},
			},
			expError: true,
		},
		{ // id too big
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("FOOBAFOOBAFOOBAFOOBAFOOBAFOOBA"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{[]byte("myBlockchainID"), bootstrap_node.URI{}},
			},
			expError: true,
		},
		{ // empty payload
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{},
			},
			expError: true,
		},
		{ // invalid payload
			token: bootstrap_node.IssueTokenMsg{
				Id:      []byte("ANY"),
				Owner:   alice.Address(),
				Details: bootstrap_node.TokenDetails{[]byte("&&&"), bootstrap_node.URI{}},
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
