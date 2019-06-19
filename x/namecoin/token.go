package namecoin

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Token{}, migration.NoModification)
}

const (
	// BucketNameToken is where we store the token definitions
	BucketNameToken = "tkn"
	// DefaultSigFigs is the default for any new token
	DefaultSigFigs = 9
)

//--- Token

var _ orm.CloneableData = (*Token)(nil)

// Validate ensures the token is valid
func (t *Token) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if !IsTokenName(t.Name) {
		return errors.Wrapf(errors.ErrInput, "invalid token name: %s", t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return errors.Wrapf(errors.ErrInput, "invalid significant figures: %d", t.SigFigs)
	}
	return nil
}

// Copy makes a new set with the same coins
func (t *Token) Copy() orm.CloneableData {
	return &Token{
		Metadata: t.Metadata.Copy(),
		Name:     t.Name,
		SigFigs:  t.SigFigs,
	}
}

// AsToken safely extracts a Token value from the object
func AsToken(obj orm.Object) *Token {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Token)
}

// AsTicker safely extracts the ticker value from the object object
func AsTicker(obj orm.Object) string {
	if obj == nil {
		return ""
	}
	return string(obj.Key())
}

// NewToken generates a new token object, using ticker as key
func NewToken(ticker, name string, sigFigs int32) orm.Object {
	value := &Token{
		Metadata: &weave.Metadata{Schema: 1},
		Name:     name,
		SigFigs:  sigFigs,
	}
	return orm.NewSimpleObj([]byte(ticker), value)
}

//--- TokenBucket - handles tokens

// TokenBucket is a type-safe wrapper around orm.BaseBucket
type TokenBucket struct {
	orm.BaseBucket
}

// NewTokenBucket initializes a TokenBucket with default name
func NewTokenBucket() TokenBucket {
	return TokenBucket{
		BaseBucket: migration.NewBucket("namecoin", BucketNameToken,
			NewToken("", "", DefaultSigFigs)),
		// orm.NewSimpleObj(nil, &Token{SigFigs: DefaultSigFigs})),
	}
}

// TODO: remove??? On afterthought, this is probably never needed
// // GetOrCreate will return the token if found, or create one
// // with the given name otherwise.
// func (b TokenBucket) GetOrCreate(db weave.KVStore, ticker string) (orm.Object, error) {
// 	obj, err := b.Get(db, ticker)
// 	if err == nil && obj == nil {
// 		obj = NewToken(ticker, "", DefaultSigFigs)
// 	}
// 	return obj, err
// }

// Get takes the token name and converts it to a byte key
func (b TokenBucket) Get(db weave.KVStore, ticker string) (orm.Object, error) {
	return b.BaseBucket.Get(db, []byte(ticker))
}

// Save enforces the proper type
func (b TokenBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Token); !ok {
		return errors.WithType(errors.ErrModel, obj.Value())
	}
	name := string(obj.Key())
	if !coin.IsCC(name) {
		return errors.Wrapf(errors.ErrInput, "invalid token name: %s", name)
	}
	return b.BaseBucket.Save(db, obj)
}

// TickerBucket can save and query Tokens (or anything with tickers...)
type TickerBucket interface {
	// GetOrCreate(db weave.KVStore, ticker string) (orm.Object, error)
	Get(db weave.KVStore, ticker string) (orm.Object, error)
	Save(db weave.KVStore, obj orm.Object) error
}
