package namecoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x/cash"
)

// BadBucket contains objects that won't satisfy Coinage interface
type BadBucket struct {
	orm.Bucket
}

func (b BadBucket) GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error) {
	// always create....
	return orm.NewSimpleObj(nil, new(Token)), nil
}

// TestValidateWalletBucket makes sure we enforce proper bucket contents
// on init.
func TestValidateWalletBucket(t *testing.T) {
	wb := NewWalletBucket()
	cb := BadBucket{orm.NewBucket("foo", orm.NewSimpleObj(nil, new(Token)))}
	// make sure this doesn't panic
	assert.NotPanics(t, func() { cash.ValidateWalletBucket(wb) })
	assert.Panics(t, func() { cash.ValidateWalletBucket(cb) })

	// make sure save errors on bad object
	db := store.MemStore()
	addr := weave.NewAddress([]byte{17, 93})
	err := wb.Save(db, orm.NewSimpleObj(addr, new(Token)))
	require.Error(t, err)
}
