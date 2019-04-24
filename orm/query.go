package orm

import "github.com/iov-one/weave"

// RegisterQuery will register a root query (literal keys)
// under "/"
func RegisterQuery(qr weave.QueryRouter) {
	// this never writes, just used to query unprefixed keys
	Bucket{}.Register("", qr)
}

// consumeIterator will read all remaining data into an
// array and close the iterator
func consumeIterator(itr weave.Iterator) ([]weave.Model, error) {
	defer itr.Close()

	var res []weave.Model
	for itr.Valid() {
		mod := weave.Model{
			Key:   itr.Key(),
			Value: itr.Value(),
		}
		res = append(res, mod)
		err := itr.Next()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
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
func queryPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([]weave.Model, error) {
	iter, err := db.Iterator(prefixRange(prefix))
	if err != nil {
		return nil, err
	}
	return consumeIterator(iter)
}
