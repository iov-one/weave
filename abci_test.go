package weave_test

import (
	"fmt"
	"testing"

	pkerr "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

func TestCreateErrorResult(t *testing.T) {
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
			assert.True(t, dres.IsErr())
			// This is if we want minimal logs in the future....
			// assert.Equal(tc.msg, dres.Log)
			assert.Contains(t, dres.Log, tc.msg)
			assert.Contains(t, dres.Log, "iov-one/weave/abci")
			assert.Equal(t, tc.code, dres.Code)

			cres := weave.CheckTxError(tc.err, true)
			assert.True(t, cres.IsErr())
			// This is if we want minimal logs in the future....
			// assert.Equal(tc.msg, cres.Log)
			assert.Contains(t, cres.Log, tc.msg)
			assert.Contains(t, cres.Log, "iov-one/weave/abci")
			assert.Equal(t, tc.code, cres.Code)
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
