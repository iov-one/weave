package currency

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &TokenInfo{}, migration.NoModification)
}

var isTokenName = regexp.MustCompile(`^[A-Za-z0-9 \-_:]{3,32}$`).MatchString

var _ orm.CloneableData = (*TokenInfo)(nil)

// NewTokenInfo returns a new instance of Token Info, as represented by orm
// object.
func NewTokenInfo(ticker, name string) orm.Object {
	return orm.NewSimpleObj([]byte(ticker), &TokenInfo{
		Metadata: &weave.Metadata{Schema: 1},
		Name:     name,
	})
}

func (t *TokenInfo) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", t.Metadata.Validate())
	if !isTokenName(t.Name) {
		errs = errors.AppendField(errs, "Name", errors.ErrState)
	}
	return errs
}

func (t *TokenInfo) Copy() orm.CloneableData {
	return &TokenInfo{
		Metadata: t.Metadata.Copy(),
		Name:     t.Name,
	}
}

// TokenInfoBucket stores TokenInfo instances, using ticker name (currency
// symbol) as the key.
type TokenInfoBucket struct {
	orm.Bucket
}

func NewTokenInfoBucket() *TokenInfoBucket {
	return &TokenInfoBucket{
		Bucket: migration.NewBucket("currency", "tokeninfo", orm.NewSimpleObj(nil, &TokenInfo{})),
	}
}

func (b *TokenInfoBucket) Get(db weave.KVStore, ticker string) (orm.Object, error) {
	return b.Bucket.Get(db, []byte(ticker))
}

func (b *TokenInfoBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*TokenInfo); !ok {
		return errors.WithType(errors.ErrModel, obj.Value())
	}
	if n := string(obj.Key()); !coin.IsCC(n) {
		return errors.Wrapf(errors.ErrCurrency, "invalid ticker: %s", n)
	}
	return b.Bucket.Save(db, obj)
}
