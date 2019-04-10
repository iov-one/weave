package client

import (
	"encoding/json"
	"fmt"

	"github.com/iov-one/tendermint/rpc/client"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

func NewRPCABCIQuery(c client.Client) Query {
	return &RPCABCIQuery{c}
}

// RPCABCIQuery exposes query interface via tendermint rpc
type RPCABCIQuery struct {
	conn client.Client
}

func (q *RPCABCIQuery) Get(key []byte) []byte {
	query, err := q.conn.ABCIQuery("/", key)
	if err != nil {
		panic(err)
	}

	resp := query.Response
	if resp.IsErr() {
		panic(fmt.Sprintf("(%d): %s", resp.Code, resp.Log))
	}

	if len(resp.Key) == 0 {
		return []byte{}
	}

	// assume there is data, parse the result sets
	var keys, vals app.ResultSet
	err = keys.Unmarshal(resp.Key)
	if err != nil {
		panic(err)
	}
	err = vals.Unmarshal(resp.Value)
	if err != nil {
		panic(err)
	}

	res, err := app.JoinResults(&keys, &vals)
	if err != nil {
		panic(err)
	}

	r, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	return r
}

func (q *RPCABCIQuery) Iterator(start, end []byte) weave.Iterator {
	panic("implement me")
}

func (q *RPCABCIQuery) ReverseIterator(start, end []byte) weave.Iterator {
	panic("implement me")
}

func NewAppABCIQuery(app abci.Application) Query {
	return &AppABCIQuery{app}
}

// AppABCIQuery exposes query interface via abci app query
type AppABCIQuery struct {
	app abci.Application
}

func (a *AppABCIQuery) Get(key []byte) []byte {
	query := a.app.Query(abci.RequestQuery{
		Path: "/",
		Data: key,
	})
	// if only the interface supported returning errors....
	if query.Code != 0 {
		panic(query.Log)
	}
	var value app.ResultSet
	if err := value.Unmarshal(query.Value); err != nil {
		panic(errors.Wrap(err, "unmarshal result set"))
	}
	if len(value.Results) == 0 {
		return nil
	}
	// TODO: assert error if len > 1 ???
	return value.Results[0]
}

// Iterator attempts to do a range iteration over the store,
// We only support prefix queries in the abci server for now.
// This client only supports listing everything...
func (a *AppABCIQuery) Iterator(start, end []byte) weave.Iterator {
	// TODO: support all prefix searches (later even more ranges)
	// look at orm/query.go:prefixRange for an idea how we turn prefix->iterator,
	// we should detect this case and reverse it so we can serialize over abci query
	if start != nil || end != nil {
		panic("iterator only implemented for entire range")
	}

	query := a.app.Query(abci.RequestQuery{
		Path: "/?prefix",
		Data: nil,
	})
	if query.Code != 0 {
		panic(query.Log)
	}
	models, err := toModels(query.Key, query.Value)
	if err != nil {
		panic(errors.Wrap(err, "cannot convert to model"))
	}

	return NewSliceIterator(models)
}

func (a *AppABCIQuery) ReverseIterator(start, end []byte) weave.Iterator {
	// TODO: load normal iterator but then play it backwards?
	panic("not implemented")
}
