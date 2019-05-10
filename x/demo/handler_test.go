package demo

import (
	"context"
	"fmt"
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	weavetest "github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestCreateAndApprove(t *testing.T) {
	create, approve, counter := testHandlers()
	assert.Equal(t, int32(0), counter.Count)

	opt, err := rawOptionsOne("Fred", 37)
	assert.Nil(t, err)

	db := store.MemStore()
	ctx := weave.WithHeight(context.Background(), 500)

	createMsg := &CreateRequestMsg{
		Metadata:  &weave.Metadata{Schema: 1},
		Title:     "FooBar",
		RawOption: opt,
	}

	res, err := create.Deliver(ctx, db, &weavetest.Tx{Msg: createMsg})
	assert.Nil(t, err)
	assert.Equal(t, int32(0), counter.Count)
	reqID := res.Data
	assert.Equal(t, 8, len(reqID))

	// load the request to ensure it looks good
	bucket := NewRequestBucket()
	obj, err := bucket.Get(db, reqID)
	assert.Nil(t, err)
	req, err := asRequest(obj)
	assert.Nil(t, err)
	assert.Equal(t, "FooBar", req.Title)
	assert.Equal(t, int32(0), req.Approvals)

	// add an approval
	approveMsg := &ApproveRequestMsg{
		Metadata:  &weave.Metadata{Schema: 1},
		RequestId: reqID,
	}
	res, err = approve.Deliver(ctx, db, &weavetest.Tx{Msg: approveMsg})
	assert.Nil(t, err)
	assert.Equal(t, int32(0), counter.Count)
	assert.Equal(t, "Approvals: 1", res.Log)

	// load to see it is updated
	bucket = NewRequestBucket()
	obj, err = bucket.Get(db, reqID)
	assert.Nil(t, err)
	req, err = asRequest(obj)
	assert.Nil(t, err)
	assert.Equal(t, "FooBar", req.Title)
	assert.Equal(t, int32(1), req.Approvals)

	// add one more approval
	res, err = approve.Deliver(ctx, db, &weavetest.Tx{Msg: approveMsg})
	assert.Nil(t, err)

	// check counter updated
	assert.Equal(t, int32(37), counter.Count)
	assert.Equal(t, "Count: 37", res.Log)

	// TODO: ensure request deleted

}

func testHandlers() (CreateRequestHandler, ApproveRequestHandler, *Counter) {
	bucket := NewRequestBucket()
	loader := LoadOptions
	counter := Counter{}
	h := CreateRequestHandler{bucket, loader}
	h2 := ApproveRequestHandler{bucket, loader, counter.Execute}
	return h, h2, &counter
}

func rawOptionsOne(name string, age int32) ([]byte, error) {
	one := &OptionOne{
		Name: name,
		Age:  age,
	}
	options := &Options{
		Option: &Options_One{
			One: one,
		},
	}
	return options.Marshal()
}

func rawOptionsTwo(data []int32) ([]byte, error) {
	two := &OptionTwo{
		Data: data,
	}
	options := &Options{
		Option: &Options_Two{
			Two: two,
		},
	}
	return options.Marshal()
}

type Counter struct {
	Count int32
}

func (c *Counter) Execute(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error) {
	switch v := msg.(type) {
	case *OptionOne:
		c.Count += v.Age
	case *OptionTwo:
		for _, d := range v.Data {
			c.Count += d
		}
	default:
		return nil, errors.Wrapf(errors.ErrInput, "unknown msg type %T", msg)
	}
	return &weave.DeliverResult{Log: fmt.Sprintf("Count: %d", c.Count)}, nil
}

var _ Executor = (*Counter)(nil).Execute
