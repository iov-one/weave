package demo

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	pathCreateRequest  = "demo/create"
	pathApproveRequest = "demo/approve"
)

var _ weave.Msg = (*CreateRequestMsg)(nil)

func (CreateRequestMsg) Path() string {
	return pathCreateRequest
}

func (o CreateRequestMsg) Validate() error {
	if err := o.Metadata.Validate(); err != nil {
		return err
	}
	if len(o.Title) < 4 {
		return errors.Wrap(errors.ErrInput, "title too short")
	}
	if len(o.RawOption) == 0 {
		return errors.Wrap(errors.ErrInput, "raw option is required")
	}
	return nil
}

var _ weave.Msg = (*ApproveRequestMsg)(nil)

func (ApproveRequestMsg) Path() string {
	return pathApproveRequest
}

func (o ApproveRequestMsg) Validate() error {
	if err := o.Metadata.Validate(); err != nil {
		return err
	}
	if len(o.RequestId) != 8 {
		return errors.Wrap(errors.ErrInput, "request id must be 8 bytes long")
	}
	return nil
}
