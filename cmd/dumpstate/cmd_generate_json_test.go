package main

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave/app"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
)

func TestExtractUsername(t *testing.T) {
	// create db store
	kv, err := bnsd.CommitKVStore("/Users/orkunkl/.mainnet-db/bns.db")
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
	for _, t := range tokens {
		fmt.Println(t)
	}
}

func TestExtractEscrow(t *testing.T) {
	// create db store
	kv, err := bnsd.CommitKVStore("/Users/orkunkl/.mainnet-db/bns.db")
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

	esc, err := extractEscrow(store)
	if err != nil {
		t.Fatalf("cannot extract usernames :%s", err)
	}
	for _, t := range esc {
		fmt.Println(t)
	}
}
