package gov

import (
	"encoding/json"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/assert"
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
	eBucket := NewElectorateBucket()
	e, err := eBucket.GetElectorate(db, weavetest.SequenceID(1))
	require.NoError(t, err)
	require.NotNil(t, e, "not found")
	assert.Equal(t, "first", e.Title)

	// and then
	rBucket := NewElectionRulesBucket()
	r, err := rBucket.GetElectionRule(db, weavetest.SequenceID(1))
	require.NoError(t, err)
	require.NotNil(t, r, "not found")
	assert.Equal(t, "foo", r.Title)

}
