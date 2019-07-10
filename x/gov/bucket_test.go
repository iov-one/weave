package gov

import (
	"reflect"
	"sort"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestHasVoted(t *testing.T) {
	bobby := weavetest.NewCondition().Address()

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
	alice := weavetest.NewCondition().Address()
	bobby := weavetest.NewCondition().Address()

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
			path: "/votes/electors",
			data: alice,
			exp:  []*Vote{&aliceVote},
		},
		"By elector bobby": {
			path: "/votes/electors",
			data: bobby,
			exp:  []*Vote{&bobbysVote},
		},
		"By unknown elector": {
			path: "/votes/electors",
			data: []byte{0x1},
			exp:  []*Vote{},
		},
		"By proposal id": {
			path: "/votes/proposals",
			data: proposalID,
			exp:  []*Vote{&aliceVote, &bobbysVote},
		},
		"By unknown proposal id": {
			path: "/votes/proposals",
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
func TestQueryElectorate(t *testing.T) {
	alice := weavetest.NewCondition().Address()
	bobby := weavetest.NewCondition().Address()

	db := store.MemStore()
	migration.MustInitPkg(db, packageName)
	vBucket := NewElectorateBucket()

	// given 2 versions
	electorateV1 := Electorate{
		Metadata:              &weave.Metadata{Schema: 1},
		Admin:                 alice,
		Title:                 "test",
		Electors:              []Elector{{Address: alice, Weight: 10}, {Address: bobby, Weight: 11}},
		TotalElectorateWeight: 21,
	}
	vRefV1, err := vBucket.Create(db, &electorateV1)
	assert.Nil(t, err)
	electorateV2 := Electorate{
		Metadata:              &weave.Metadata{Schema: 1},
		Admin:                 alice,
		Title:                 "test",
		Electors:              []Elector{{Address: alice, Weight: 20}, {Address: bobby, Weight: 22}},
		TotalElectorateWeight: 42,
		Version:               1,
	}
	vRefV2, err := vBucket.Update(db, vRefV1.ID, &electorateV2)
	assert.Nil(t, err)

	specs := map[string]struct {
		path      string
		queryMode string
		data      []byte
		exp       []Electorate
	}{
		"By ID and first version": {
			path: "/electorates",
			data: mustMarshal(t, vRefV1),
			exp:  []Electorate{electorateV1},
		},
		"By ID and second version": {
			path: "/electorates",
			data: mustMarshal(t, vRefV2),
			exp:  []Electorate{electorateV2},
		},
		"By ID and unknown version": {
			path: "/electorates",
			data: mustMarshal(t, &orm.VersionedIDRef{ID: vRefV1.ID, Version: 99}),
		},
		"By prefix query and ID": {
			path:      "/electorates",
			queryMode: weave.PrefixQueryMod,
			data:      mustMarshal(t, &orm.VersionedIDRef{ID: vRefV1.ID}),
			exp:       []Electorate{electorateV1, electorateV2},
		},
		"By unknown ID": {
			path: "/electorates",
			data: mustMarshal(t, &orm.VersionedIDRef{ID: []byte{0x1}, Version: 1}),
		},
		"By unknown key": {
			path: "/electorates",
			data: []byte{0x1},
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
			models, err := h.Query(db, spec.queryMode, spec.data)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(spec.exp) == 0 && len(models) == 0 {
				return
			}
			receivedElectorate := make([]Electorate, len(models))
			for i, v := range models {
				obj, err := vBucket.Parse(nil, v.Value)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				e, err := asElectorate(obj)
				if err != nil {
					t.Fatalf("Unknown type: %T", obj)
				}
				receivedElectorate[i] = *e
			}
			assert.Equal(t, spec.exp, receivedElectorate)
		})
	}
}

func byAddrSorter(src []*Vote) func(i int, j int) bool {
	byAddressSorter := func(i, j int) bool {
		return src[i].Elector.Address.String() < src[j].Elector.Address.String()
	}
	return byAddressSorter
}

type marshaller interface {
	Marshal() ([]byte, error)
}

func mustMarshal(t *testing.T, m marshaller) []byte {
	b, err := m.Marshal()
	assert.Nil(t, err)
	return b
}
