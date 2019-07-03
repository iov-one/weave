package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/msgfee"
)

func TestCmdSendTokensHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-src", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-dst", "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
		"-amount", "5 DOGE",
		"-memo", "a memo",
	}
	if err := cmdSendTokens(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new token transfer transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*cash.SendMsg)

	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(msg.Source))
	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(msg.Destination))
	assert.Equal(t, "a memo", msg.Memo)
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), msg.Amount)
}

func TestCmdWithFeeHappyPath(t *testing.T) {
	sendMsg := &cash.SendMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Source:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
		Destination: fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
		Amount:      coin.NewCoinp(5, 0, "DOGE"),
		Memo:        "a memo",
	}
	sendTx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: sendMsg,
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, sendTx); err != nil {
		t.Fatalf("cannot serialize transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-payer", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-amount", "5 DOGE",
	}
	if err := cmdWithFee(&input, &output, args); err != nil {
		t.Fatalf("cannot attach a fee to transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}
	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(tx.Fees.Payer))
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), tx.Fees.Fees)

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	// Message must be unmodified.
	assert.Equal(t, sendMsg, txmsg)
}

func TestCmdWithFeeHappyPathDefaultAmount(t *testing.T) {
	sendMsg := &cash.SendMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Source:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
		Destination: fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
		Amount:      coin.NewCoinp(5, 0, "DOGE"),
		Memo:        "a memo",
	}
	sendTx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: sendMsg,
		},
	}

	cases := map[string]struct {
		Conf    cash.Configuration
		Fees    map[string]coin.Coin
		WantFee *coin.Coin
	}{
		"only minimal fee": {
			Conf: cash.Configuration{
				Metadata:   &weave.Metadata{Schema: 1},
				MinimalFee: coin.NewCoin(4, 0, "BTC"),
			},
			Fees:    nil,
			WantFee: coin.NewCoinp(4, 0, "BTC"),
		},
		"only message fee": {
			Conf: cash.Configuration{
				Metadata:   &weave.Metadata{Schema: 1},
				MinimalFee: coin.NewCoin(0, 0, ""),
			},
			Fees: map[string]coin.Coin{
				sendMsg.Path(): coin.NewCoin(17, 0, "IOV"),
			},
			WantFee: coin.NewCoinp(17, 0, "IOV"),
		},
		"custom message fee is more important than global setting": {
			Conf: cash.Configuration{
				Metadata:   &weave.Metadata{Schema: 1},
				MinimalFee: coin.NewCoin(123, 0, "IOV"),
			},
			Fees: map[string]coin.Coin{
				sendMsg.Path(): coin.NewCoin(11, 0, "IOV"),
			},
			WantFee: coin.NewCoinp(11, 0, "IOV"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var input bytes.Buffer
			if _, err := writeTx(&input, sendTx); err != nil {
				t.Fatalf("cannot serialize transaction: %s", err)
			}

			tm := newCashConfTendermintServer(t, tc.Conf, tc.Fees)
			defer tm.Close()

			var output bytes.Buffer
			args := []string{
				// Instead of providing an amount, rely on what is configured
				// for the network.
				"-tm", tm.URL,
			}
			if err := cmdWithFee(&input, &output, args); err != nil {
				t.Fatalf("cannot attach a fee to transaction: %s", err)
			}

			tx, _, err := readTx(&output)
			if err != nil {
				t.Fatalf("cannot unmarshal created transaction: %s", err)
			}
			assert.Equal(t, tc.WantFee, tx.Fees.Fees)
		})
	}
}

type abciQueryRequest struct {
	Method string `json:"method"`
	Params struct {
		Data string `json:"data"` // hex-encoded
		Path string `json:"path"`
	} `json:"params"`
}

// newCashConfTendermintServer returns an HTTP server that can respond to an
// HTTP json-rpc request with given configuration.
func newCashConfTendermintServer(
	t *testing.T,
	conf cash.Configuration,
	msgfees map[string]coin.Coin,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected tendermint request: %s", r.URL)
		}

		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		var req abciQueryRequest
		err := decoder.Decode(&req)
		assert.Nil(t, err)
		assert.Equal(t, "abci_query", req.Method)
		assert.Equal(t, "/", req.Params.Path)

		raw, err := hex.DecodeString(req.Params.Data)
		assert.Nil(t, err)

		if bytes.Equal(raw, []byte("_c:cash")) {
			io.WriteString(w, tmResponse(t, raw, &conf))
			return
		}

		if bytes.HasPrefix(raw, []byte("schema:msgfee")) {
			pkg := "schema:msgfee"
			cnt := raw[len(pkg):]
			if bytes.Equal(cnt, []byte{0, 0, 0, 1}) {
				schema := &migration.Schema{
					Metadata: &weave.Metadata{Schema: 1},
					Pkg:      pkg,
					Version:  1,
				}
				io.WriteString(w, tmResponse(t, raw, schema))
			} else {
				io.WriteString(w, tmEmptyResponse(t))
			}
			return
		}

		if bytes.HasPrefix(raw, []byte("msgfee:")) {
			path := string(raw[len("msgfee:"):])
			fee, ok := msgfees[path]
			if !ok {
				io.WriteString(w, tmEmptyResponse(t))
				return
			}
			io.WriteString(w, tmResponse(t, raw, &msgfee.MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  path,
				Fee:      fee,
			}))
			return
		}

		t.Fatalf("unexpected tendermint request: %X", raw)
	}))
}

// tmResponse returns a tenderming HTTP response for a configuration query.
// Returned response does not contain "key" or "height" information.
func tmResponse(t testing.TB, key []byte, payload interface{ Marshal() ([]byte, error) }) string {
	value, err := payload.Marshal()
	assert.Nil(t, err)

	model := []weave.Model{{Key: key, Value: value}}

	keySet, err := app.ResultsFromKeys(model).Marshal()
	assert.Nil(t, err)
	encKey := base64.StdEncoding.EncodeToString(keySet)

	valSet, err := app.ResultsFromValues(model).Marshal()
	assert.Nil(t, err)
	encVal := base64.StdEncoding.EncodeToString(valSet)

	return `{
	  "jsonrpc": "2.0",
	  "id": "",
	  "result": {
	    "response": {
	      "key": "` + encKey + `",
	      "value": "` + encVal + `"
	    }
	  }
	}`
}

func tmEmptyResponse(t testing.TB) string {
	set, err := (&app.ResultSet{}).Marshal()
	assert.Nil(t, err)
	enc := base64.StdEncoding.EncodeToString(set)

	return `{
	  "jsonrpc": "2.0",
	  "id": "",
	  "result": {
	    "response": {
	      "key": "` + enc + `",
	      "value": "` + enc + `"
	    }
	  }
	}`
}
