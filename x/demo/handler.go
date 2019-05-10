package demo

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	createRequestCost  = 100
	approveRequestCost = 50

	requiredApprovals = 2
)

// CreateRequestHandler stores an initial request
type CreateRequestHandler struct {
	bucket RequestBucket
	loader OptionLoader
}

var _ weave.Handler = CreateRequestHandler{}

// Check does the validation and sets the cost of the transaction
func (h CreateRequestHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}

	res := &weave.CheckResult{
		GasAllocated: createRequestCost,
	}
	return res, nil
}

// Deliver moves the tokens from sender to the swap account if all conditions are met.
func (h CreateRequestHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}

	// create a swap object
	request := &Request{
		Metadata:  msg.Metadata,
		Title:     msg.Title,
		RawOption: msg.RawOption,
		Approvals: 0,
	}

	obj, err := h.bucket.Create(db, request)
	if err != nil {
		return nil, err
	}

	// return id of request to use in future calls
	res := &weave.DeliverResult{
		Data: obj.Key(),
	}
	return res, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateRequestHandler) validate(ctx weave.Context, tx weave.Tx) (*CreateRequestMsg, error) {
	var msg CreateRequestMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// validate the content
	option, err := h.loader(msg.RawOption)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse the raw_options field")
	}
	if err := option.Validate(); err != nil {
		return nil, errors.Wrap(err, "request options are invalid")
	}

	return &msg, nil
}

// ApproveRequestHandler will add an approval to an existing request,
// executing it when it hits needed approvals
type ApproveRequestHandler struct {
	bucket   RequestBucket
	loader   OptionLoader
	executor Executor
}

var _ weave.Handler = ApproveRequestHandler{}

// Check does the validation and sets the cost of the transaction
func (h ApproveRequestHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}

	res := &weave.CheckResult{
		GasAllocated: approveRequestCost,
	}
	return res, nil
}

// Deliver moves the tokens from sender to the swap account if all conditions are met.
func (h ApproveRequestHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, tx)
	if err != nil {
		return nil, err
	}

	// load the request
	obj, err := h.bucket.Get(db, msg.RequestId)
	if err != nil {
		return nil, err
	}
	request, err := asRequest(obj)
	if err != nil {
		return nil, err
	}

	// update approvals and save until we hit threshold
	request.Approvals++
	if request.Approvals < requiredApprovals {
		err := h.bucket.Save(db, obj)
		return &weave.DeliverResult{}, err
	}

	// here we will execute it
	opt, err := h.loader(request.RawOption)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse loaded request options")
	}
	return h.executor(ctx, db, opt)
}

// validate does all common pre-processing between Check and Deliver.
func (h ApproveRequestHandler) validate(ctx weave.Context, tx weave.Tx) (*ApproveRequestMsg, error) {
	var msg ApproveRequestMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}
