package weave_test

import (
	"fmt"
	"strconv"
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

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		dres := weave.DeliverTxError(tc.err)
		assert.True(dres.IsErr(), i)
		assert.Equal(tc.msg, dres.Log, i)
		assert.Equal(tc.code, dres.Code, i)

		cres := weave.CheckTxError(tc.err)
		assert.True(cres.IsErr(), i)
		assert.Equal(tc.msg, cres.Log, i)
		assert.Equal(tc.code, cres.Code, i)
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
	assert.Equal(t, gas, ac.Gas)
	assert.Equal(t, int64(0), ac.Fee)
	assert.Empty(t, ac.Data)
}
