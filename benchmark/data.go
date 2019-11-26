package benchmark

import (
	"github.com/iov-one/weave/orm"
)

func (d Data) Validate() error {
	return nil
}

func dataKey(name, domain string) []byte {
	key := make([]byte, 0, len(name)+len(domain)+1)
	key = append(key, domain...)
	key = append(key, '*')
	key = append(key, name...)
	return key
}

func newDataBucket() orm.ModelBucket {
	return orm.NewModelBucket("data", &Data{})
}
