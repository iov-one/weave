package weave_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/confio/weave"
)

func TestVersion(t *testing.T) {
	weave.GitCommit = ""
	assert.Equal(t, "v0.1.0", weave.Version())

	weave.GitCommit = "12345678"
	assert.Equal(t, "v0.1.0 12345678", weave.Version())

}
