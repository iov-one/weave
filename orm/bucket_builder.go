package orm

import (
	"fmt"
)

type BucketBuilderOpt interface {
	Apply(*BucketBuilder) *BucketBuilder
}

type BucketBuilderOptFunc func(*BucketBuilder) *BucketBuilder

func (f BucketBuilderOptFunc) Apply(bb *BucketBuilder) *BucketBuilder {
	return f(bb)
}

type BucketBuilder struct {
	b   bucket
	pkg string
}

func NewBucketBuilder(name string, model Cloneable, opts ...BucketBuilderOpt) *BucketBuilder {
	if !isBucketName(name) {
		panic(fmt.Sprintf("Illegal bucket: %s", name))
	}

	b := bucket{
		name:   name,
		prefix: append([]byte(name), ':'),
		proto:  model,
	}
	bb := &BucketBuilder{
		b: b,
	}
	for _, v := range opts {
		bb = v.Apply(bb)
	}
	return bb
}

func (b *BucketBuilder) WithIndex(name string, indexer Indexer, unique bool) *BucketBuilder {
	return b.WithMultiKeyIndex(name, asMultiKeyIndexer(indexer), unique)
}

func (b *BucketBuilder) WithMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool) *BucketBuilder {
	b.b = b.b.withMultiKeyIndex(name, indexer, unique)
	return b
}

func (b BucketBuilder) Build() BaseBucket {
	return b.b
}
