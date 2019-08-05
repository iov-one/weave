package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
)

// TestCmdSubmitTxHappyPath will set fees, sign the tx, and submit it... ensuring the
// whole workflow is valid.
func TestCmdSubmitTxHappyPath(t *testing.T) {
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Source:      fromHex(t, addr),
				Destination: fromHex(t, addr),
				Amount: &coin.Coin{
					Whole:  5,
					Ticker: "IOV",
				},
			},
		},
	}

	var input bytes.Buffer
	if _, err := writeTx(&input, tx); err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}

	var withFee bytes.Buffer
	feeArgs := []string{
		"-tm", tmURL,
	}
	err := cmdWithFee(&input, &withFee, feeArgs)
	assert.Nil(t, err)

	// we must sign it with a key we have
	var signedTx bytes.Buffer
	signArgs := []string{
		"-tm", tmURL,
		"-key", mustCreateFile(t, bytes.NewReader(fromHex(t, privKeyHex))),
	}
	err = cmdSignTransaction(&withFee, &signedTx, signArgs)
	assert.Nil(t, err)

	var output bytes.Buffer
	args := []string{
		"-tm", tmURL,
	}
	if err := cmdSubmitTransaction(&signedTx, &output, args); err != nil {
		t.Fatalf("cannot submit the transaction: %s", err)
	}
}

func TestSubmitTxResponse(t *testing.T) {
	fmts := map[string]func([]byte) (string, error){
		"mymsg":      fmtSequence,
		"anothermsg": func(b []byte) (string, error) { return string(b), nil },
	}

	cases := map[string]struct {
		Tx          weave.Tx
		DeliverData []byte
		WantResp    []string
		WantErr     bool
	}{
		"not supported message returns no response": {
			Tx:          &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "message/not/supported"}},
			DeliverData: []byte(`this-data-must-not-be-parsed`),
			WantResp:    nil,
		},
		"a sequence formatter returns a decimal representation": {
			Tx: &weavetest.Tx{
				Msg: &weavetest.Msg{
					RoutePath: "mymsg", // Registered in fmts
				},
			},
			DeliverData: weavetest.SequenceID(123456),
			WantResp:    []string{"123456"},
		},
		"formatter for the invalid response returns an error": {
			Tx: &weavetest.Tx{
				Msg: &weavetest.Msg{
					RoutePath: "mymsg", // Registered in fmts
				},
			},
			DeliverData: []byte("x"),
			WantResp:    nil,
			WantErr:     true,
		},
		"a batch message response is parsed and every response is formatted separately": {
			Tx: &weavetest.Tx{
				Msg: &batchMsg{
					Msgs: []weave.Msg{
						&weavetest.Msg{RoutePath: "mymsg"},
						&weavetest.Msg{RoutePath: "anothermsg"},
					},
				},
			},
			DeliverData: batchResp(t,
				weavetest.SequenceID(123456),
				[]byte("foobar"),
			),
			WantResp: []string{
				"123456",
				"foobar",
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			resp, err := extractResponse(tc.Tx, tc.DeliverData, fmts)
			hasErr := err != nil
			if tc.WantErr != hasErr {
				t.Fatalf("returned error: %+v", err)
			}
			assert.Equal(t, tc.WantResp, resp)
		})
	}
}

// batchMsg clubs together any number of messages and implements batch.Msg
// interface. It does not intent to implement weave.Msg interface though.
type batchMsg struct {
	weave.Msg
	weave.Batch

	Msgs []weave.Msg
}

func (m *batchMsg) MsgList() ([]weave.Msg, error) {
	return m.Msgs, nil
}

// batchResp returns a set of response serialized using batch byte array. This
// is the same format as the batch extension is using for serializing multiple
// responses.
func batchResp(t testing.TB, responses ...[]byte) []byte {
	t.Helper()
	arr := batch.ByteArrayList{
		Elements: responses,
	}
	b, err := arr.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal byte array list: %s", err)
	}
	return b
}
