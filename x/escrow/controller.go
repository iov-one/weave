package escrow

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/cash"
)

type controller struct {
	cash   cash.Controller
	bucket Bucket
}

func NewController(cash cash.Controller, bucket Bucket) *controller {
	return &controller{
		cash:   cash,
		bucket: bucket,
	}
}

// Deposit transfers the given amounts from source wallet to the escrow account and persist it.
func (m *controller) Deposit(db weave.KVStore, escrow *Escrow, escrowID []byte, src weave.Address, amounts coin.Coins) error {
	available := coin.Coins(escrow.Amount).Clone()
	err := m.moveCoins(db, src, Condition(escrowID).Address(), amounts)
	if err != nil {
		return err
	}
	for _, c := range amounts {
		available, err = available.Add(*c)
		if err != nil {
			return err
		}
	}
	escrow.Amount = available
	return m.bucket.Save(db, orm.NewSimpleObj(escrowID, escrow))
}

// Deposit transfers the given amounts from escrow account to dest wallet and persist it.
// If no coins are remaining in the escrow account it is deleted.
func (m *controller) Withdraw(db weave.KVStore, escrow *Escrow, escrowID []byte, dest weave.Address, amounts coin.Coins) error {
	available := coin.Coins(escrow.Amount).Clone()
	err := m.moveCoins(db, Condition(escrowID).Address(), dest, amounts)
	if err != nil {
		return err
	}
	// remove coin from remaining balance
	for _, c := range amounts {
		available, err = available.Subtract(*c)
		if err != nil {
			return err
		}
	}
	escrow.Amount = available
	// if there is something left, just update the balance...
	if available.IsPositive() {
		return m.bucket.Save(db, orm.NewSimpleObj(escrowID, escrow))
	}
	// otherwise we finished the escrow and can delete it
	return m.bucket.Delete(db, escrowID)
}

func (m *controller) moveCoins(db weave.KVStore, src weave.Address, dest weave.Address, amounts coin.Coins) error {
	for _, c := range amounts {
		err := m.cash.MoveCoins(db, src, dest, *c)
		if err != nil {
			// this will rollback the half-finished tx
			return err
		}
	}
	return nil
}
