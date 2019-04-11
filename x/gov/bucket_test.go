package gov

import (
	"testing"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestVoteBucket(t *testing.T) {
	db := store.MemStore()
	vBucket := NewVoteBucket()
	proposalID := weavetest.SequenceID(1)
	obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Signature: bobby, Weight: 10}})
	if err := vBucket.Save(db, obj); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// when
	voted, err := vBucket.HasVoted(db, proposalID, bobby)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if exp, got := true, voted; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
}
