package gov

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/require"
)

func TestInitFromGenesis(t *testing.T) {
	const genesisSnippet = `
{
  "governance": {
    "electorate": [
      {
        "title": "first",
        "electors": [
          {
            "weight": 10,
            "signature": "1111111111111111111111111111111111111111"
          },
          {
            "weight": 11,
            "signature": "2222222222222222222222222222222222222222"
          }
        ]
      },
      {
        "title": "second",
        "electors": [
          {
            "weight": 1,
            "signature": "3333333333333333333333333333333333333333"
          }
        ]
      }
    ],
    "rules": [
      {
        "title": "fooo",
        "voting_period_hours": 1,
        "fraction": {
          "numerator": 2,
          "denominator": 3
        }
      },
      {
        "title": "barr",
        "voting_period_hours": 2,
        "fraction": {
          "numerator": 1,
          "denominator": 2
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
	// first electorate ok
	e, err := NewElectorateBucket().GetElectorate(db, weavetest.SequenceID(1))
	if err != nil || e == nil {
		t.Fatalf("unexpected result: error: %s", err)
	}
	if exp, got := "first", e.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := 2, len(e.Electors); exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := addr("1111111111111111111111111111111111111111"), e.Electors[0].Signature; !exp.Equals(got) {
		t.Errorf("expected %X but got %X", exp, got)
	}
	if exp, got := uint32(10), e.Electors[0].Weight; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := addr("2222222222222222222222222222222222222222"), e.Electors[1].Signature; !exp.Equals(got) {
		t.Errorf("expected %X but got %X", exp, got)
	}
	if exp, got := uint32(11), e.Electors[1].Weight; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	// second electorate ok
	e, err = NewElectorateBucket().GetElectorate(db, weavetest.SequenceID(2))
	if err != nil || e == nil {
		t.Fatalf("unexpected result: error: %s", err)
	}
	if exp, got := "second", e.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := 1, len(e.Electors); exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := addr("3333333333333333333333333333333333333333"), e.Electors[0].Signature; !exp.Equals(got) {
		t.Errorf("expected %X but got %X", exp, got)
	}
	if exp, got := uint32(1), e.Electors[0].Weight; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}

	// and then
	// first election rule ok
	r, err := NewElectionRulesBucket().GetElectionRule(db, weavetest.SequenceID(1))
	if err != nil || r == nil {
		t.Fatalf("unexpected result: error: %s", err)
	}
	if got, exp := "fooo", r.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := uint32(1), r.VotingPeriodHours; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := (Fraction{Numerator: 2, Denominator: 3}), r.Threshold; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	// second election rule ok
	r, err = NewElectionRulesBucket().GetElectionRule(db, weavetest.SequenceID(2))
	if err != nil || r == nil {
		t.Fatalf("unexpected result: error: %s", err)
	}
	if got, exp := "barr", r.Title; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := uint32(2), r.VotingPeriodHours; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
	if exp, got := (Fraction{Numerator: 1, Denominator: 2}), r.Threshold; exp != got {
		t.Errorf("expected %v but got %v", exp, got)
	}
}

func addr(s string) weave.Address {
	a, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return a
}
