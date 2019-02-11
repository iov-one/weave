package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMultisig(t *testing.T) {
	// Create, sign, view and submit a multisignature request flow. This is
	// not a unit test neither an integration test. It is nevertheless
	// helpful.
	tm := newTendermintServer(t)
	defer tm.Close()

	var out bytes.Buffer
	args := []string{
		"-power", "7",
		"-pubkey", "j4JRVstX",
		"-multisig", "5AE2C58796B0AD48FFE7602EAC3353488C859A2B",
	}
	if err := cmdMultisigNew(nil, &out, args); err != nil {
		t.Fatalf("cannot create a multisig request: %s", err)
	}
	unsignedReq := out.Bytes()
	out.Reset()
	if err := cmdMultisigView(bytes.NewReader(unsignedReq), &out, nil); err != nil {
		t.Fatalf("cannot view a multisig request: %s", err)
	}
	t.Logf("unsigned request: %s", out.String())

	out.Reset()
	args = []string{
		"-tm", tm.URL,
		"-key", "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9",
	}
	if err := cmdMultisigSign(bytes.NewReader(unsignedReq), &out, args); err != nil {
		t.Fatalf("cannot sign a multisig request: %s", err)
	}

	signedReq := out.Bytes()
	out.Reset()
	if err := cmdMultisigView(bytes.NewReader(signedReq), &out, nil); err != nil {
		t.Fatalf("cannot view a multisig request: %s", err)
	}
	t.Logf("signed request: %s", out.String())

	out.Reset()
	args = []string{
		"-tm", tm.URL,
	}
	if err := cmdMultisigSubmit(bytes.NewReader(signedReq), &out, args); err != nil {
		t.Fatalf("cannot submit a multisig request: %s", err)
	}
}

func newTendermintServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *logRequestFl {
			b, _ := ioutil.ReadAll(r.Body)
			r.Body = ioutil.NopCloser(bytes.NewReader(b))
			t.Logf("tendermint request: %s %s: %s", r.Method, r.URL.Path, string(b))
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/genesis":
			io.WriteString(w, `
				{
					"jsonrpc": "2.0",
					"id": "",
					"result": {
						"genesis": {
							"chain_id": "test-chain-ZIYjN0",
							"validators": [],
							"app_state": {
								"gconf": {
									"cash:minimal_fee": {},
									"cash:collector_address": "0000000000000000000000000000000000000000"
								},
								"update_validators": {
									"addresses": [
										"b1ca7e78f74423ae01da3b51e676934d9105f282",
										"E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"
									]
								},
								"multisig": [
								{
									"sigs": [
										"b1ca7e78f74423ae01da3b51e676934d9105f282",
										"E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"
									],
									"admin_threshold": 2,
									"activation_threshold": 1
								}
								]
							}
						}
					}
				}
			`)
		case r.Method == "POST" && r.URL.Path == "/":
			// This is an RPC call - response always depends on the
			// submitted content. For our tests it does not matter
			// that much what is returned.
			io.WriteString(w, `
				{
					"jsonrpc": "2.0",
					"id": "jsonrpc-client",
					"result": {"response": {"height": "12345"}}
				}
			`)

		default:
			http.Error(w, "not implemented", http.StatusNotImplemented)
		}
	}))
}

var logRequestFl = flag.Bool("logrequest", false, "Log all requests send to tendermint mock server. This is useful when writing new test. Use curl to send the same request to a real tendermint node and record the response.")
