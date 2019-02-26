package currency

import (
	"regexp"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var isTokenName = regexp.MustCompile(`^[A-Za-z0-9 \-_:]{3,32}$`).MatchString

var _ orm.CloneableData = (*TokenInfo)(nil)

// NewTokenInfo returns a new instance of Token Info, as represented by orm
// object.
func NewTokenInfo(ticker, name string) orm.Object {
	return orm.NewSimpleObj([]byte(ticker), &TokenInfo{
		Name: name,
	})
}

func (t *TokenInfo) Validate() error {
	if !isTokenName(t.Name) {
		return errors.ErrInvalidState.Newf("invalid token name %v", t.Name)
	}
	return nil
}

func (t *TokenInfo) Copy() orm.CloneableData {
	return &TokenInfo{
		Name: t.Name,
	}
}

// TockenInfoBucket stores TokenInfo instances, using ticker name (currency
// symbol) as the key.
type TokenInfoBucket struct {
	orm.Bucket
}

func NewTokenInfoBucket() *TokenInfoBucket {
	return &TokenInfoBucket{
		Bucket: orm.NewBucket("tokeninfo", orm.NewSimpleObj(nil, &TokenInfo{})),
	}
}

func (b *TokenInfoBucket) Get(db weave.KVStore, ticker string) (orm.Object, error) {
	return b.Bucket.Get(db, []byte(ticker))
}

func (b *TokenInfoBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*TokenInfo); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	if n := string(obj.Key()); !coin.IsCC(n) {
		return coin.ErrInvalidCurrency.New(n)
	}
	return b.Bucket.Save(db, obj)
}
