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

func TestCreateResult(t *testing.T) {
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
