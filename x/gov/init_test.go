package gov

import (
	"encoding/json"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitFromGenesis(t *testing.T) {
	const genesisSnippet = `
{
  "governance": {
    "electorate": [
      {
        "title": "first",
        "participants": [
          {
            "weight": 10,
            "signature": "E94323317C46BDA2268FA3698BAF4F95B893E8C7"
          },
          {
            "weight": 11,
            "signature": "FE5526DE08337DFEF5CF45EF3ED8C577B854DE34"
          }
        ]
      }
    ],
    "rules": [
      {
        "title": "foo",
        "voting_period_hours": 1,
        "fraction": {
          "numerator": 2,
          "denominator": 3
        }
      }
    ]
  }
}`
	var opts weave.Options
	require.NoError(t, json.Unmarshal([]byte(genesisSnippet), &opts))

	db := store.MemStore()
	// when
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}
	// then
	e, err := NewElectorateBucket().GetElectorate(db, weavetest.SequenceID(1))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if e == nil {
		t.Fatal("must not be nil")
	}
	if exp, got := "first", e.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if got, exp := 2, len(e.Participants); exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}

	// and then
	r, err := NewElectionRulesBucket().GetElectionRule(db, weavetest.SequenceID(1))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if r == nil {
		t.Fatal("must not be nil")
	}
	if got, exp := "foo", r.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
}
