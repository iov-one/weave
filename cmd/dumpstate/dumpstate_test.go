package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/iov-one/weave/app"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
)

func TestDumpState(t *testing.T) {
	// create db store
	kv, err := bnsd.CommitKVStore("./testdata/bns.db")
	if err != nil {
		t.Fatalf("cannot initialize bnsd commit store: %s", err)
	}
	// set db version/height
	version := 54397
	err = kv.LoadVersion(int64(version))
	if err != nil {
		t.Fatalf("cannot load db version: %s", err)
	}
	store := app.NewCommitStore(kv)

	tokens, err := extractUsername(store)
	if err != nil {
		t.Fatalf("cannot extract usernames :%s", err)
	}
	want, err := os.Open("username.gold.json")
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONEqual(t, want, tokens)

	esc, err := extractEscrow(store)
	if err != nil {
		t.Fatalf("cannot extract escrows: %s", err)
	}
	want, err = os.Open("escrow.gold.json")
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONEqual(t, want, esc)

	cont, err := extractContracts(store)
	if err != nil {
		t.Fatalf("cannot extract contracts: %s", err)
	}
	want, err = os.Open("contract.gold.json")
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONEqual(t, want, cont)

	cash, err := extractWallets(store)
	if err != nil {
		t.Fatalf("cannot extract wallets: %s", err)
	}
	want, err = os.Open("cash.gold.json")
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONEqual(t, want, cash)
}

func AssertJSONEqual(t testing.TB, want io.Reader, got interface{}) {
	t.Helper()

	var w json.RawMessage
	if err := json.NewDecoder(want).Decode(&w); err != nil {
		t.Fatalf("cannot decode JSON serialized body: %s", err)
	}
	var g json.RawMessage
	g, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("cannot decode JSON serialized body: %s", err)
	}

	w1, _ := json.MarshalIndent(w, "", "    ")
	g1, _ := json.MarshalIndent(g, "", "    ")

	if !bytes.Equal(w1, g1) {
		t.Logf("want JSON:\n%s", w1)
		t.Logf("got JSON:\n%s", g1)
		t.Fatal("unexpected result")
	}
}

/*
// create db store
func TestGenerateGold(t *testing.T) {
	kv, err := bnsd.CommitKVStore("./testdata/bns.db")
	if err != nil {
		t.Fatalf("cannot initialize bnsd commit store: %s", err)
	}
	// set db version/height
	version := 54397
	err = kv.LoadVersion(int64(version))
	if err != nil {
		t.Fatalf("cannot load db version: %s", err)
	}
	store := app.NewCommitStore(kv)

	tokens, err := extractWallets(store)
	if err != nil {
		t.Fatalf("cannot extract usernames :%s", err)
	}

	outFile, err := os.Create("cash.gold.json")
	if err != nil {
		t.Fatalf("cannot extract usernames :%s", err)
	}

	enc := json.NewEncoder(outFile)
	enc.SetIndent("", "\t")
	if err := enc.Encode(tokens); err != nil {
		fmt.Printf("cannot write to file: %s\n", err)
		os.Exit(1)
	}
}
*/
