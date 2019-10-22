package gconf

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

func TestLoadSave(t *testing.T) {
	db := store.MemStore()
	c := configuration{raw: "foobar"}
	if err := Save(db, "gconf", &c); err != nil {
		t.Fatalf("cannot save configuration: %s", err)
	}
	if err := Load(db, "gconf", &c); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
}

// configuration is a mock of a protobuf configuration object. It does not
// marshal/unmarshal itself properly but rather ensures that the right bytes
// were passed around.
type configuration struct {
	err error
	raw string
}

func (c *configuration) Marshal() ([]byte, error) {
	return []byte(c.raw), c.err
}

func (c *configuration) Unmarshal(raw []byte) error {
	if !bytes.Equal([]byte(c.raw), raw) {
		return fmt.Errorf("expected %q, got %q", c.raw, raw)
	}
	return c.err
}

func (c *configuration) Validate() error {
	return c.err
}

func TestConfModelBucket(t *testing.T) {
	db := store.MemStore()
	b := NewConfigurationModelBucket()

	if err := b.Has(db, []byte("does-not-exist")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("expected configuration to not be found: %+v", err)
	}

	c := &configuration{raw: "foobar"}
	if _, err := b.Put(db, []byte("mymod"), c); err != nil {
		t.Fatalf("cannot store configuration: %s", err)
	}

	if err := b.One(db, []byte("mymod"), c); err != nil {
		t.Fatalf("cannot get configuration: %s", err)
	}
	if err := b.Has(db, []byte("mymod")); err != nil {
		t.Fatalf("mymod configuration should be present: %s", err)
	}

	// Using Load/Save should work interchangeably
	if err := Load(db, "mymod", c); err != nil {
		t.Fatalf("cannot load: %s", err)
	}
	c2 := &configuration{raw: "second conf"}
	if err := Save(db, "mymod", c2); err != nil {
		t.Fatalf("cannot save: %s", err)
	}
	if err := b.One(db, []byte("mymod"), c2); err != nil {
		t.Fatalf("cannot get configuration: %s", err)
	}
}
