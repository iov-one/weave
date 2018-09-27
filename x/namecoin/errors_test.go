package namecoin

import (
	"regexp"
	"testing"

	"github.com/iov-one/weave/crypto"
	"github.com/stretchr/testify/require"
)

func TestErrNoSuchWallet(t *testing.T) {
	hasNonASCII := regexp.MustCompile("[[:^ascii:]]").MatchString
	addr := crypto.GenPrivKeyEd25519().PublicKey().Address()
	msg := ErrNoSuchWallet(addr).Error()
	require.False(t, hasNonASCII(msg), msg)
}
