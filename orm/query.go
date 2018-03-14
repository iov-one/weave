package orm

import "github.com/confio/weave"

// consumeIterator will read all remaining data into an
// array and close the iterator
func consumeIterator(itr weave.Iterator) []weave.Model {
	defer itr.Close()

	var res []weave.Model
	for ; itr.Valid(); itr.Next() {
		mod := weave.Model{
			Key:   itr.Key(),
			Value: itr.Value(),
		}
		res = append(res, mod)
	}
	return res
}

// prefixRange turns a prefix into (start, end) to create
// and iterator
func prefixRange(prefix []byte) ([]byte, []byte) {
	// special case: no prefix is whole range
	if len(prefix) == 0 {
		return nil, nil
	}

	// copy the prefix and update last byte
	end := make([]byte, len(prefix))
	copy(end, prefix)
	l := len(end) - 1
	end[l]++

	// wait, what if that overflowed?....
	for end[l] == 0 && l > 0 {
		l--
		end[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && end[0] == 0 {
		end = nil
	}
	return prefix, end
}

// queryPrefix returns a prefix query as Models
func queryPrefix(db weave.ReadOnlyKVStore, prefix []byte) []weave.Model {
	return consumeIterator(db.Iterator(prefixRange(prefix)))
}
