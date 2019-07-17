package multisig

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestGenesisKey(t *testing.T) {
	// to generate signature addresses, use
	//   openssl rand -hex 20
	const genesis = `
		{
			"multisig": [
				{
					"participants": [
						{"weight": 1, "signature": "e4c7e4c71a3b301a2521753ddd1d2c26fd6fe1bf"},
						{"weight": 2, "signature": "904bc35e341b428d4faa535022b553efbc443d49"},
						{"weight": 7, "signature": "91d66344d78599b66e1b504db958b1b07a8f5049"}
					],
					"activation_threshold": 2,
					"admin_threshold": 3
				}
			]
		}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	migration.MustInitPkg(db, "multisig")
	var ini Initializer
	if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewContractBucket()
	var c Contract
	if err := bucket.One(db, weavetest.SequenceID(1), &c); err != nil {
		t.Fatalf("cannot fetch contract information: %s", err)
	}
	if want, got := Weight(2), c.ActivationThreshold; want != got {
		t.Errorf("want activation threshold %d, got %d", want, got)
	}
	if want, got := Weight(3), c.AdminThreshold; want != got {
		t.Errorf("want admin threshold %d, got %d", want, got)
	}
	wantParticipants := []*Participant{
		{Weight: 1, Signature: fromHex(t, "e4c7e4c71a3b301a2521753ddd1d2c26fd6fe1bf")},
		{Weight: 2, Signature: fromHex(t, "904bc35e341b428d4faa535022b553efbc443d49")},
		{Weight: 7, Signature: fromHex(t, "91d66344d78599b66e1b504db958b1b07a8f5049")},
	}
	if !reflect.DeepEqual(wantParticipants, c.Participants) {
		t.Errorf("want participants \n%#v\n, got \n%#v", wantParticipants, c.Participants)
	}

}

func fromHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("cannot decode %q hex encoded data: %s", s, err)
	}
	return raw
}
