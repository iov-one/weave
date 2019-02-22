package namecoin

import (
	"github.com/iov-one/weave/errors"
	"regexp"
	"testing"

	"github.com/iov-one/weave/crypto"
	"github.com/stretchr/testify/require"
)

func TestErrNoSuchWallet(t *testing.T) {
	hasNonASCII := regexp.MustCompile("[[:^ascii:]]").MatchString
	addr := crypto.GenPrivKeyEd25519().PublicKey().Address()
	msg := errors.ErrNotFound.Newf("wallet %s", addr).Error()
	require.False(t, hasNonASCII(msg), msg)
}
