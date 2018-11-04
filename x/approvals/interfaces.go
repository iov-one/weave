package approvals

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

type Approvable interface {
	orm.CloneableData
	GetOwner() []byte
	GetApprovals() [][]byte
	UpdateApprovals(approvals [][]byte)
}

type AddApprovalMsg interface {
	weave.Msg
	GetId() []byte
	GetApproval() []byte
}

func validate(msg AddApprovalMsg) error {
	return weave.Condition(msg.GetApproval()).Validate()
}
