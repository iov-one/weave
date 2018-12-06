package currency

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

var _ orm.CloneableData = (*TokenInfo)(nil)

func (t *TokenInfo) Validate() error {
	if !isTokenName(t.Name) {
		return ErrInvalidTokenName(t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return ErrInvalidSigFigs(t.SigFigs)
	}
	return nil
}

var isTokenName = regexp.MustCompile(`^[A-Za-z0-9 \-_:]{3,32}$`).MatchString

const (
	minSigFigs = 0
	maxSigFigs = 9
)

func (t *TokenInfo) Copy() orm.CloneableData {
	return &TokenInfo{
		Name:    t.Name,
		SigFigs: t.SigFigs,
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
		return ErrInvalidObject(obj.Value())
	}
	if n := string(obj.Key()); !x.IsCC(n) {
		return x.ErrInvalidCurrency(n)
	}
	return b.Bucket.Save(db, obj)
}
