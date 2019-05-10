package demo

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

func (r Request) Validate() error {
	if err := r.GetMetadata().Validate(); err != nil {
		return errors.Wrap(err, "request metadata")
	}
	if len(r.Title) < 4 {
		return errors.Wrap(errors.ErrState, "request needs title fo 4 characters or longer")
	}
	if len(r.RawOption) == 0 {
		return errors.Wrap(errors.ErrState, "request requires raw_options to be set")
	}
	return nil
}

func (r *Request) Copy() orm.CloneableData {
	return &Request{
		Metadata:  r.Metadata.Copy(),
		Title:     r.Title,
		Approvals: r.Approvals,
		RawOption: append([]byte{}, r.RawOption...),
	}
}

// RequestBucket is the persistent bucket for request objects.
type RequestBucket struct {
	orm.IDGenBucket
}

// NewRequestBucket returns a bucket for managing request.
func NewRequestBucket() *RequestBucket {
	b := orm.NewBucket("requests", orm.NewSimpleObj(nil, &Request{}))
	return &RequestBucket{
		IDGenBucket: orm.WithSeqIDGenerator(b, "id"),
	}
}

func asRequest(obj orm.Object) (*Request, error) {
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	res, ok := obj.Value().(*Request)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return res, nil
}
