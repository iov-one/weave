package weavetest

import (
	"context"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
)

func TestAuthNoSigners(t *testing.T) {
	var a Auth

	if got := a.GetConditions(nil); got != nil {
		t.Fatalf("unexpected conditions: %+v", got)
	}

	if a.HasAddress(nil, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}

func TestAuthUsingSignerAndSigners(t *testing.T) {
	conds := []weave.Condition{
		NewCondition(),
		NewCondition(),
		NewCondition(),
	}

	a := Auth{
		Signer:  conds[2],
		Signers: conds[:2],
	}

	if got := a.GetConditions(nil); !reflect.DeepEqual(got, conds) {
		for i, c := range got {
			t.Logf("condition %d: %s", i, c)
		}
		t.Fatalf("unexpected conditions")
	}

	for i, c := range conds {
		if !a.HasAddress(nil, c.Address()) {
			t.Errorf("condition %d (%s) address should be present", i, c)
		}
	}

	if a.HasAddress(nil, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}

func TestAuthUsingSigner(t *testing.T) {
	a := Auth{Signer: NewCondition()}

	if got := a.GetConditions(nil); len(got) != 1 || !got[0].Equals(a.Signer) {
		t.Fatalf("unexpected conditions: %+v", got)
	}

	if !a.HasAddress(nil, a.Signer.Address()) {
		t.Error("signer condition should be present")
	}

	if a.HasAddress(nil, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}

func TestAuthUsingSigners(t *testing.T) {
	conds := []weave.Condition{
		NewCondition(),
		NewCondition(),
		NewCondition(),
	}
	a := Auth{Signers: conds}

	if got := a.GetConditions(nil); !reflect.DeepEqual(got, conds) {
		for i, c := range got {
			t.Logf("condition %d: %s", i, c)
		}
		t.Fatalf("unexpected conditions")
	}

	for i, c := range conds {
		if !a.HasAddress(nil, c.Address()) {
			t.Errorf("condition %d (%s) address should be present", i, c)
		}
	}

	if a.HasAddress(nil, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}

func TestCtxAuth(t *testing.T) {
	perms := []weave.Condition{
		NewCondition(),
		NewCondition(),
	}
	ctx := context.Background()

	a := CtxAuth{Key: "auth"}
	ctx = a.SetConditions(ctx, perms...)

	if got := a.GetConditions(ctx); !reflect.DeepEqual(got, perms) {
		for i, c := range got {
			t.Logf("condition %d: %s", i, c)
		}
		t.Fatal("unexpected conditions")
	}

	for i, p := range perms {
		if !a.HasAddress(ctx, p.Address()) {
			t.Errorf("condition %d (%s) address should be present", i, p)
		}
	}

	if a.HasAddress(ctx, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}

func TestCtxAuthEmptyContext(t *testing.T) {
	ctx := context.Background()
	a := CtxAuth{Key: "auth"}
	if got := a.GetConditions(ctx); got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
	if a.HasAddress(ctx, NewCondition().Address()) {
		t.Fatal("random condition must not be present")
	}
}
