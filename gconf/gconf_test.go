package gconf

import (
	"bytes"
	"fmt"
	"testing"

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
