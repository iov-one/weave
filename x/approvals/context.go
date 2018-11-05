package approvals

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the multisig module

const (
	contextKeyApprovals contextKey = iota
	NoCount                        = -1
	NoTimeout                      = -1
)

// withApproval is a private method, as only this module
// can add a approved signer
func withApproval(ctx weave.Context, id []byte) weave.Context {
	val, _ := ctx.Value(contextKeyApprovals).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextKeyApprovals, []weave.Condition{ApprovalCondition(id, "usage")})
	}

	return context.WithValue(ctx, contextKeyApprovals, append(val, ApprovalCondition(id, "usage")))
}

// ApprovalCondition returns condition for a given action and signer
func ApprovalCondition(id []byte, approvalType string) weave.Condition {
	return weave.NewCondition("approval", approvalType, id)
}

func ApprovalConditionWithCount(id []byte, approvalType string, count int64) weave.Condition {
	key := bytes.Join([][]byte{id, encode(count), encode(NoTimeout)}, []byte(":"))
	return weave.NewCondition("approval", approvalType, key)
}

func ApprovalConditionWithTimeout(id []byte, approvalType string, timeout int64) weave.Condition {
	key := bytes.Join([][]byte{id, encode(NoCount), encode(timeout)}, []byte(":"))
	return weave.NewCondition("approval", approvalType, key)
}

func ApprovalConditionWithCountAndTimeout(id []byte, approvalType string, count, timeout int64) weave.Condition {
	key := bytes.Join([][]byte{id, encode(count), encode(timeout)}, []byte(":"))
	return weave.NewCondition("approval", approvalType, key)
}

func decode(bz []byte) int64 {
	if bz == nil {
		return 0
	}
	val := binary.BigEndian.Uint64(bz)
	return int64(val)
}

func encode(val int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(val))
	return bz
}

// Authenticate gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyApprovals).([]weave.Condition)
	if val == nil {
		return nil
	}
	return val
}

// HasAddress returns true iff this address is in GetConditions
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}

func HasApprovals(ctx weave.Context, auth x.Authenticator, action string, conditions [][]byte, owner []byte) (bool, []weave.Condition) {
	var approved []weave.Condition
	if auth.HasAddress(ctx, ApprovalCondition(owner, "usage").Address()) {
		return true, approved
	}

	height, _ := weave.GetHeight(ctx)
	for _, cond := range conditions {
		_, act, addr, _ := weave.Condition(cond).Parse()
		if act == action {
			target, count, timeout := ApprovalKey(addr).Parse()
			if count == 0 || (timeout != NoTimeout && height > timeout) {
				continue
			}
			if auth.HasAddress(ctx, ApprovalCondition(target, "usage").Address()) {
				approved = append(approved, cond)
			}
		}
	}
	return len(approved) > 0, approved
}

func Approve(ctx weave.Context, auth x.Authenticator, action string, appr Approvable) bool {
	ok, used := HasApprovals(ctx, auth, action, appr.GetApprovals(), appr.GetOwner())
	if !ok {
		return false
	}

	conditions := appr.GetApprovals()
	for _, u := range used {
		for idx, a := range conditions {
			if u.Equals(weave.Condition(a)) {
				conditions[idx] = withUpdatedCount(u)
			}
		}
	}

	return true
}

func withUpdatedCount(cond weave.Condition) weave.Condition {
	_, action, addr, _ := cond.Parse()
	target, count, timeout := ApprovalKey(addr).Parse()
	if count != NoCount {
		return ApprovalConditionWithCountAndTimeout(target, action, count-1, timeout)
	}
	return cond
}

type ApprovalKey []byte

func (k ApprovalKey) Parse() ([]byte, int64, int64) {
	key := bytes.Split(k, []byte(":"))
	switch len(key) {
	case 1:
		return k, NoCount, NoTimeout
	case 3:
		return key[0], decode(key[1]), decode(key[2])
	default:
		return nil, NoCount, NoTimeout
	}
}
