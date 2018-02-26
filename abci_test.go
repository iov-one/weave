package weave_test

import (
	"fmt"
	"strings"
	"testing"

	pkerr "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

func TestCreateErrorResult(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		err  error
		msg  string
		code uint32
	}{
		{fmt.Errorf("base"), "base", errors.CodeInternalErr},
		{pkerr.New("dave"), "dave", errors.CodeInternalErr},
		{errors.New("nonce", errors.CodeUnauthorized), "nonce", errors.CodeUnauthorized},
		{errors.Wrap(fmt.Errorf("wrap")), "wrap", errors.CodeInternalErr},
		{errors.WithCode(fmt.Errorf("no sender"), errors.CodeUnrecognizedAddress), "no sender", errors.CodeUnrecognizedAddress},
		{errors.ErrDecoding(), errors.ErrDecoding().Error(), errors.CodeTxParseError},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {

			dres := weave.DeliverTxError(tc.err)
			assert.True(dres.IsErr())
			// This is if we want minimal logs in the future....
			// assert.Equal(tc.msg, dres.Log)
			assert.True(strings.HasPrefix(dres.Log, tc.msg))
			assert.Contains(dres.Log, "github.com/confio/weave")
			assert.Equal(tc.code, dres.Code)

			cres := weave.CheckTxError(tc.err)
			assert.True(cres.IsErr())
			// This is if we want minimal logs in the future....
			// assert.Equal(tc.msg, cres.Log)
			assert.True(strings.HasPrefix(cres.Log, tc.msg))
			assert.Contains(cres.Log, "github.com/confio/weave")
			assert.Equal(tc.code, cres.Code)
		})
	}
}

func TestCreateResults(t *testing.T) {
	d, msg := []byte{1, 3, 4}, "got it"
	dres := weave.DeliverResult{Data: d, Log: msg}
	ad := dres.ToABCI()
	assert.EqualValues(t, d, ad.Data)
	assert.Equal(t, msg, ad.Log)
	assert.Empty(t, ad.Tags)

	c, gas := "aok", int64(12345)
	cres := weave.NewCheck(gas, c)
	ac := cres.ToABCI()
	assert.Equal(t, c, ac.Log)
	assert.Equal(t, gas, ac.GasWanted)
	assert.Equal(t, int64(0), ac.Fee.Value)
	assert.Empty(t, ac.Data)
}
