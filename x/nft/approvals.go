package nft

import (
	"time"

	"github.com/iov-one/weave"
)

type Approvals []*Approval

func (a Approvals) Clone() Approvals {
	if a == nil {
		return nil
	}
	o := make([]*Approval, len(a))
	for i, v := range a {
		o[i] = v.Clone()
	}
	return o
}
func (a Approvals) ByAction(action ActionKind) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.Action == action {
			r = append(r, v)
		}
	}
	return r
}
func (a Approvals) ByAddress(to weave.Address) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.ToAccountAddress().Equals(to) {
			r = append(r, v)
		}
	}
	return r
}
func (a Approvals) Remove(obsoletes ...*Approval) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		found := false
		for _, o := range obsoletes { // todo: revisit
			if v.Equals(o) {
				found = true
				break
			}
		}
		if !found {
			r = append(r, v)
		}
	}
	return r
}

func (a Approvals) AsValues() []Approval {
	r := make([]Approval, 0)
	for _, v := range a {
		r = append(r, *v)
	}
	return r
}

func (a Approvals) WithoutExpired() Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.Options.Timeout != 0 && time.Unix(0, v.Options.Timeout).Before(time.Now()) {
			continue
		}
		if v.Options.Count == 0 {
			continue
		}
		r = append(r, v)
	}
	return r
}

func (a Approvals) Exists() bool {
	return len(a) != 0
}
