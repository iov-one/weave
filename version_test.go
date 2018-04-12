package bov_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	bov "github.com/iov-one/bcp-demo"
)

func TestVersion(t *testing.T) {
	bov.GitCommit = ""
	assert.Equal(t, "v0.1.0", bov.Version())

	bov.GitCommit = "12345678"
	assert.Equal(t, "v0.1.0 12345678", bov.Version())
}
