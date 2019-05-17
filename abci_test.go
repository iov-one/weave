package weave

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestCreateResults(t *testing.T) {
	dres := DeliverResult{
		Data: []byte{1, 3, 4},
		Log:  "got it",
	}
	ad := dres.ToABCI()
	assert.EqualValues(t, dres.Data, ad.Data)
	assert.Equal(t, dres.Log, ad.Log)
	assert.Empty(t, ad.Tags)

	log, gas := "aok", int64(12345)
	cres := NewCheck(gas, log)
	ac := cres.ToABCI()
	assert.Equal(t, log, ac.Log)
	assert.Equal(t, gas, ac.GasWanted)
	assert.Empty(t, ac.Data)
	assert.Empty(t, ac.Data)

}

func TestDeliverTxError(t *testing.T) {
	cases := map[string]struct {
		err      error
		debug    bool
		wantResp abci.ResponseDeliverTx
	}{
		"internal error is hidden": {
			err:   fmt.Errorf("cannot connect to the database"),
			debug: false,
			wantResp: abci.ResponseDeliverTx{
				Code: 1,
				Log:  "cannot deliver tx: internal error",
			},
		},
		"internal error is not hidden when in debug mode": {
			err:   fmt.Errorf("cannot connect to the database"),
			debug: true,
			wantResp: abci.ResponseDeliverTx{
				Code: 1,
				Log:  "cannot deliver tx: cannot connect to the database",
			},
		},
		"weave error is exposed": {
			err:   errors.Wrap(notFoundErr{}, "not here"),
			debug: false,
			wantResp: abci.ResponseDeliverTx{
				Code: 666,
				Log:  "cannot deliver tx: not here: not found",
			},
		},
		"weave error is exposed in debug mode": {
			err:   errors.Wrap(notFoundErr{}, "not here"),
			debug: false,
			wantResp: abci.ResponseDeliverTx{
				Code: 666,
				Log:  "cannot deliver tx: not here: not found",
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			resp := DeliverTxError(tc.err, tc.debug)
			if !reflect.DeepEqual(resp, tc.wantResp) {
				t.Fatalf("unexpected response: %+v", resp)
			}
		})
	}
}

func TestCheckTxError(t *testing.T) {
	cases := map[string]struct {
		err      error
		debug    bool
		wantResp abci.ResponseCheckTx
	}{
		"internal error is hidden": {
			err:   fmt.Errorf("cannot connect to the database"),
			debug: false,
			wantResp: abci.ResponseCheckTx{
				Code: 1,
				Log:  "cannot check tx: internal error",
			},
		},
		"internal error is not hidden when in debug mode": {
			err:   fmt.Errorf("cannot connect to the database"),
			debug: true,
			wantResp: abci.ResponseCheckTx{
				Code: 1,
				Log:  "cannot check tx: cannot connect to the database",
			},
		},
		"abci error is exposed": {
			err:   errors.Wrap(notFoundErr{}, "not here"),
			debug: false,
			wantResp: abci.ResponseCheckTx{
				Code: 666,
				Log:  "cannot check tx: not here: not found",
			},
		},
		"weave error is exposed in debug mode": {
			err:   errors.Wrap(notFoundErr{}, "not here"),
			debug: false,
			wantResp: abci.ResponseCheckTx{
				Code: 666,
				Log:  "cannot check tx: not here: not found",
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			resp := CheckTxError(tc.err, tc.debug)
			if !reflect.DeepEqual(resp, tc.wantResp) {
				t.Fatalf("unexpected response: %+v", resp)
			}
		})
	}
}

// notFoundErr is a custom implementation of an error that provides an ABCICode
// method.
type notFoundErr struct{}

func (notFoundErr) ABCICode() uint32 { return 666 }

func (notFoundErr) Error() string { return "not found" }
