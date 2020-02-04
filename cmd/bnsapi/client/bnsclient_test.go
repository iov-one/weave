package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/iov-one/weave/cmd/bnsapi/util"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

func TestABCIKeyQuery(t *testing.T) {
	// Run a fake Tendermint API server that will answer to only expected
	// query requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/abci_query" {
			t.Fatalf("unexpected path: %q", r.URL)
		}
		q := r.URL.Query()
		switch {
		case q.Get("path") == `"/myentity"` && q.Get("data") == `"entitykey"`:
			writeServerResponse(t, w, [][]byte{
				[]byte("0001"),
			}, []weave.Persistent{
				&persistentMock{Raw: []byte("content")},
			})
		case q.Get("path") == `"/myentity"`:
			writeServerResponse(t, w, nil, nil)
		default:
			t.Fatalf("unknown condition: %q", r.URL)
		}

	}))
	defer srv.Close()

	bns := NewHTTPBnsClient(srv.URL)

	dest := persistentMock{Raw: []byte("content")}
	if err := ABCIKeyQuery(context.Background(), bns, "/myentity", []byte("entitykey"), &dest); err != nil {
		t.Fatalf("cannot get by key: %v", err)
	}

	if err := ABCIKeyQuery(context.Background(), bns, "/myentity", []byte("xxxxx"), &dest); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBnsClientDo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/foo" {
			t.Fatalf("unexpected path: %q", r.URL)
		}
		_, _ = io.WriteString(w, `
			{
				"result": "a result"
			}
		`)
	}))
	defer srv.Close()

	bns := NewHTTPBnsClient(srv.URL)

	var result string
	if err := bns.Get(context.Background(), "/foo", &result); err != nil {
		t.Fatalf("get: %s", err)
	}
	if result != "a result" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestABCIFullRangeQuery(t *testing.T) {
	hexit := func(s string) string {
		return hex.EncodeToString([]byte(s))
	}

	// Run a fake Tendermint API server that will answer to only expected
	// query requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/abci_query" {
			t.Fatalf("unexpected path: %q", r.URL)
		}
		q := r.URL.Query()
		switch {
		case q.Get("path") == `"/myquery?range"` && q.Get("data") == `""`:
			writeServerResponse(t, w, [][]byte{
				[]byte("0001"),
				[]byte("0002"),
				[]byte("0003"),
			}, []weave.Persistent{
				&persistentMock{Raw: []byte("1")},
				&persistentMock{Raw: []byte("2")},
				&persistentMock{Raw: []byte("3")},
			})
		case q.Get("path") == `"/myquery?range"` && q.Get("data") == `"`+hexit("0003")+`:"`:
			writeServerResponse(t, w, [][]byte{
				[]byte("0003"), // Filter is inclusive.
				[]byte("0004"),
			}, []weave.Persistent{
				&persistentMock{Raw: []byte("3")},
				&persistentMock{Raw: []byte("4")},
			})
		case q.Get("path") == `"/myquery?range"` && q.Get("data") == `"`+hexit("0004")+`:"`:
			writeServerResponse(t, w, [][]byte{
				[]byte("0004"), // Filter is inclusive.
			}, []weave.Persistent{
				&persistentMock{Raw: []byte("4")},
			})
		default:
			t.Logf("query: %q", q.Get("query"))
			t.Logf("data: %q", q.Get("data"))
			t.Errorf("not supported request: %q", r.URL)
			http.Error(w, "not supported", http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	bns := NewHTTPBnsClient(srv.URL)
	it := ABCIFullRangeQuery(context.Background(), bns, "/myquery", "")

	var keys [][]byte
consumeIterator:
	for {
		switch key, err := it.Next(ignoreModel{}); {
		case err == nil:
			keys = append(keys, key)
		case errors.ErrIteratorDone.Is(err):
			break consumeIterator
		default:
			t.Fatalf("iterator failed: %s", err)
		}

	}

	// ABCIFullRangeQuery iterator must return all available keys in the
	// right order and each key only once. We do not check values because
	// we ignore them in this test.
	wantKeys := [][]byte{
		[]byte("0001"),
		[]byte("0002"),
		[]byte("0003"),
		[]byte("0004"),
	}

	if !reflect.DeepEqual(wantKeys, keys) {
		for i, k := range keys {
			t.Logf("key %2d: %q", i, k)
		}
		t.Fatalf("unexpected %d keys", len(keys))
	}
}

// ignoreModel is a stub. Its unmarshal is a no-op. Use it together with an
// iterator if you do not care about the result unloading.
type ignoreModel struct {
	orm.Model
}

func (ignoreModel) Unmarshal([]byte) error { return nil }

func writeServerResponse(t testing.TB, w http.ResponseWriter, keys [][]byte, models []weave.Persistent) {
	t.Helper()

	k, v := util.SerializePairs(t, keys, models)

	// Minimal acceptable by our code jsonrpc response.
	type dict map[string]interface{}
	payload := dict{
		"result": dict{
			"response": dict{
				"key":   k,
				"value": v,
			},
		},
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("cannot write response: %s", err)
	}

}

type persistentMock struct {
	orm.Model

	Raw []byte
	Err error
}

func (m *persistentMock) Unmarshal(raw []byte) error {
	if m.Raw != nil && !bytes.Equal(m.Raw, raw) {
		return fmt.Errorf("want %q, got %q", m.Raw, raw)
	}
	m.Raw = raw
	return m.Err
}

func (m *persistentMock) Marshal() ([]byte, error) {
	return m.Raw, m.Err
}
