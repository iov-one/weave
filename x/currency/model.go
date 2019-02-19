package currency

import (
	"github.com/iov-one/weave/errors"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	minSigFigs = 0
	maxSigFigs = 9
)

var isTokenName = regexp.MustCompile(`^[A-Za-z0-9 \-_:]{3,32}$`).MatchString

var _ orm.CloneableData = (*TokenInfo)(nil)

// NewTokenInfo returns a new instance of Token Info, as represented by orm
// object.
func NewTokenInfo(ticker, name string, sigFigs int32) orm.Object {
	return orm.NewSimpleObj([]byte(ticker), &TokenInfo{
		Name:    name,
		SigFigs: sigFigs,
	})
}

func (t *TokenInfo) Validate() error {
	if !isTokenName(t.Name) {
		return ErrInvalidTokenName(t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return ErrInvalidSigFigs(t.SigFigs)
	}
	return nil
}

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
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	if n := string(obj.Key()); !x.IsCC(n) {
		return x.ErrInvalidCurrency.New(n)
	}
	return b.Bucket.Save(db, obj)
}
