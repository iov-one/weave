package gconf

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestSave(t *testing.T) {
	db := store.MemStore()

	conf := MyConfig{
		Number: 55,
		Text:   "foo & bar",
		Addr:   weavetest.RandomAddr(t),
		secret: "foo",
	}

	if err := Save(db, conf); err != nil {
		t.Fatalf("cannot save configuration: %s", err)
	}
}

type MyConfig struct {
	Number int64
	Text   string
	Addr   weave.Address
	secret string
}
