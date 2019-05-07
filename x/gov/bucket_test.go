package gov

import (
	"reflect"
	"sort"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestHasVoted(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, packageName)
	vBucket := NewVoteBucket()
	proposalID := weavetest.SequenceID(1)
	obj := vBucket.Build(db, proposalID,
		Vote{
			Metadata: &weave.Metadata{Schema: 1},
			Voted:    VoteOption_Yes,
			Elector:  Elector{Address: bobby, Weight: 10},
		},
	)
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

func TestQueryVotes(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, packageName)
	proposalID := weavetest.SequenceID(1)
	vBucket := NewVoteBucket()

	// given
	bobbysVote := Vote{
		Metadata: &weave.Metadata{Schema: 1},
		Voted:    VoteOption_Yes,
		Elector:  Elector{Address: bobby, Weight: 1},
	}
	aliceVote := Vote{
		Metadata: &weave.Metadata{Schema: 1},
		Voted:    VoteOption_No,
		Elector:  Elector{Address: alice, Weight: 10},
	}
	for _, v := range []Vote{bobbysVote, aliceVote} {
		obj := vBucket.Build(db, proposalID, v)
		if err := vBucket.Save(db, obj); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	specs := map[string]struct {
		path string
		data []byte
		exp  []*Vote
	}{
		"By elector alice": {
			path: "/vote/elector",
			data: alice,
			exp:  []*Vote{&aliceVote},
		},
		"By elector bobby": {
			path: "/vote/elector",
			data: bobby,
			exp:  []*Vote{&bobbysVote},
		},
		"By unknown elector": {
			path: "/vote/elector",
			data: []byte{0x1},
			exp:  []*Vote{},
		},
		"By proposal id": {
			path: "/vote/proposal",
			data: proposalID,
			exp:  []*Vote{&aliceVote, &bobbysVote},
		},
		"By unknown proposal id": {
			path: "/vote/proposal",
			data: []byte{0x1},
			exp:  []*Vote{},
		},
	}

	qr := weave.NewQueryRouter()
	RegisterQuery(qr)
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			h := qr.Handler(spec.path)
			if h == nil {
				t.Fatal("must not be nil")
			}
			models, err := h.Query(db, "", spec.data)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			receivedVotes := make([]*Vote, len(models))
			for i, v := range models {
				obj, err := vBucket.Parse(nil, v.Value)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				var ok bool
				receivedVotes[i], ok = obj.Value().(*Vote)
				if !ok {
					t.Fatalf("Unknown type: %T", obj)
				}
			}
			// returned list is not always in the same order. sort both lists
			// to prevent flickering tests
			sort.Slice(spec.exp, byAddrSorter(receivedVotes))
			sort.Slice(spec.exp, byAddrSorter(spec.exp))
			if exp, got := spec.exp, receivedVotes; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %#v but got %#v", exp, got)
			}

		})
	}
}

func byAddrSorter(src []*Vote) func(i int, j int) bool {
	byAddressSorter := func(i, j int) bool {
		return src[i].Elector.Address.String() < src[j].Elector.Address.String()
	}
	return byAddressSorter
}
