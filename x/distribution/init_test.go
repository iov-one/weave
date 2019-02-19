package distribution

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestGenesisKey(t *testing.T) {
	const genesis = `
		{
			"distribution": [
				{
					"admin": "E94323317C46BDA2268FA3698BAF4F95B893E8C7",
					"recipients": [
						{"weight": 2, "address": "E94323317C46BDA2268FA3698BAF4F95B893E8C7"},
						{"weight": 1, "address": "FE5526DE08337DFEF5CF45EF3ED8C577B854DE34"}
					]
				}
			]
		}
	`
	addr1, _ := hex.DecodeString("E94323317C46BDA2268FA3698BAF4F95B893E8C7")
	addr2, _ := hex.DecodeString("FE5526DE08337DFEF5CF45EF3ED8C577B854DE34")

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewRevenueBucket()
	rev, err := bucket.GetRevenue(db, asSeqID(1))
	if err != nil {
		t.Fatalf("cannot fetch revenue: %s", err)
	} else if rev == nil {
		t.Fatal("revenue not found")
	}

	if !rev.Admin.Equals(addr1) {
		t.Fatalf("unexpected admin address: %q", rev.Admin)
	}
	if n := len(rev.Recipients); n != 2 {
		t.Fatalf("expected one recipient, got %d", n)
	}
	if r := rev.Recipients[0]; r.Weight != 2 {
		t.Fatalf("want weight 2, got %d", r.Weight)
	}
	if r := rev.Recipients[0]; !r.Address.Equals(addr1) {
		t.Fatalf("unexected address: %q", r.Address)
	}
	if r := rev.Recipients[1]; r.Weight != 1 {
		t.Fatalf("want weight 1, got %d", r.Weight)
	}
	if r := rev.Recipients[1]; !r.Address.Equals(addr2) {
		t.Fatalf("unexected address: %q", r.Address)
	}
}
