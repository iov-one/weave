package orm

import "github.com/confio/weave"

// ConsumeIterator will read all remaining data into an
// array and close the iterator
func ConsumeIterator(itr weave.Iterator) []weave.Model {
	defer itr.Close()

	res := []weave.Model{}
	for ; itr.Valid(); itr.Next() {
		mod := weave.Model{
			Key:   itr.Key(),
			Value: itr.Value(),
		}
		res = append(res, mod)
	}
	return res
}
