package weave_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/stretchr/testify/assert"
)

func TestCreateResults(t *testing.T) {
	dres := weave.DeliverResult{
		Data: []byte{1, 3, 4},
		Log:  "got it",
	}
	ad := dres.ToABCI()
	assert.EqualValues(t, dres.Data, ad.Data)
	assert.Equal(t, dres.Log, ad.Log)
	assert.Empty(t, ad.Tags)

	log, gas := "aok", int64(12345)
	cres := weave.NewCheck(gas, log)
	ac := cres.ToABCI()
	assert.Equal(t, log, ac.Log)
	assert.Equal(t, gas, ac.GasWanted)
	assert.Empty(t, ac.Data)
}
