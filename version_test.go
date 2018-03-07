package bov_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	bov "github.com/iov-one/bov-core"
)

func TestVersion(t *testing.T) {
	bov.GitCommit = ""
	assert.Equal(t, "v0.1.0-dev", bov.Version())

	bov.GitCommit = "12345678"
	assert.Equal(t, "v0.1.0-dev 12345678", bov.Version())
}
