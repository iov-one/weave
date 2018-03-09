package namecoin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/confio/weave/x/cash"
)

func TestWalletBucket(t *testing.T) {

	bucket := NewWalletBucket()
	// make sure this doesn't panic
	assert.NotPanics(t, func() { cash.ValidateWalletBucket(bucket) })
}
