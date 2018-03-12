package namecoin

import (
	"github.com/confio/weave"
	"github.com/confio/weave/orm"
)

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
	if !IsTokenName(t.Name) {
		return ErrInvalidTokenName(t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return ErrInvalidSigFigs(t.SigFigs)
	}
	return nil
}

// Copy makes a new set with the same coins
func (t *Token) Copy() orm.CloneableData {
	return &Token{
		Name:    t.Name,
		SigFigs: t.SigFigs,
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
		Name:    name,
		SigFigs: sigFigs,
	}
	return orm.NewSimpleObj([]byte(ticker), value)
}

//--- TokenBucket - handles tokens

// TokenBucket is a type-safe wrapper around orm.Bucket
type TokenBucket struct {
	orm.Bucket
}

// NewTokenBucket initializes a TokenBucket with default name
func NewTokenBucket() TokenBucket {
	return TokenBucket{
		Bucket: orm.NewBucket(BucketNameToken,
			NewToken("", "", DefaultSigFigs)),
	}
}

// GetOrCreate will return the token if found, or create one
// with the given name otherwise.
func (b TokenBucket) GetOrCreate(db weave.KVStore, ticker string) (orm.Object, error) {
	obj, err := b.Get(db, ticker)
	if err == nil && obj == nil {
		obj = NewToken(ticker, "", DefaultSigFigs)
	}
	return obj, err
}

// Get takes the token name and converts it to a byte key
func (b TokenBucket) Get(db weave.KVStore, ticker string) (orm.Object, error) {
	return b.Bucket.Get(db, []byte(ticker))
}

// TickerBucket can save and query Tokens (or anything with tickers...)
type TickerBucket interface {
	GetOrCreate(db weave.KVStore, ticker string) (orm.Object, error)
	Get(db weave.KVStore, ticker string) (orm.Object, error)
	Save(db weave.KVStore, obj orm.Object) error
}
