package multisig

import "github.com/iov-one/weave"

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it in the
// database.
func (*Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	var contracts []struct {
		Participants []struct {
			Signature weave.Address `json:"signature"`
			Weight    Weight        `json:"weight"`
		} `json:"participants"`
		ActivationThreshold Weight `json:"activation_threshold"`
		AdminThreshold      Weight `json:"admin_threshold"`
	}
	if err := opts.ReadOptions("multisig", &contracts); err != nil {
		return err
	}

	bucket := NewContractBucket()
	for _, c := range contracts {
		ps := make([]*Participant, 0, len(c.Participants))
		for _, p := range c.Participants {
			ps = append(ps, &Participant{
				Signature: p.Signature,
				Weight:    p.Weight,
			})
		}
		contract := Contract{
			Metadata:            &weave.Metadata{Schema: 1},
			Participants:        ps,
			ActivationThreshold: c.ActivationThreshold,
			AdminThreshold:      c.AdminThreshold,
		}
		obj, err := bucket.Build(db, &contract)
		if err != nil {
			return err
		}
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
