package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iov-one/weave/x/escrow"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/multisig"
)

const prefix = "iov"

type Out struct {
	Username []tokenFormat    `json:"username"`
	Wallets  []genesisAccount `json:"cash"`
	Escrow   []escrowFormat   `json:"escrow"`
	Contract []contractFormat `json:"contract"`
}

type genesisAccount struct {
	Address string `json:"address"`
	cash.Set
}

type tokenFormat struct {
	Username string
	Targets  []username.BlockchainAddress
	Owner    string
}

type escrowFormat struct {
	Source      string         `json:"source"`
	Arbiter     string         `json:"arbiter"`
	Destination string         `json:"destination"`
	Timeout     weave.UnixTime `json:"timeout"`
	Amount      []*coin.Coin   `json:"amount"`
	Address     string         `json:"address"`
}

type contractFormat struct {
	Participants        []*multisig.Participant `json:"participants"`
	ActivationThreshold multisig.Weight         `json:"activation_threshold"`
	AdminThreshold      multisig.Weight         `json:"admin_threshold"`
	Address             weave.Address           `json:"address"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Export state data. Pipe-in app version as input.`)
		flag.PrintDefaults()
	}
	var (
		dbFl = flag.String("db", env("BNSD_DB_DIR", os.ExpandEnv("$HOME")+"/.bns"),
			"bnsd database directory")
		heightFl = flag.Uint("height", 0,
			"commit height")
		outFl = flag.String("out", "./dump.json",
			"dump output directory")
	)

	flag.Parse()

	if *heightFl == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// validate db
	dbPath := filepath.Join(*dbFl, "bns.db")
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		fmt.Printf("db file does not exists: %s\n", err)
		os.Exit(1)
	}

	// create db store
	kv, err := bnsd.CommitKVStore(dbPath)
	if err != nil {
		fmt.Printf("cannot initialize bnsd commit store: %s\n", err)
		os.Exit(1)
	}

	store := app.NewCommitStore(kv)
	// set db version/height
	err = kv.LoadVersion(int64(*heightFl))
	if err != nil {
		fmt.Printf("cannot load db version: %s\n", err)
		os.Exit(1)
	}

	// create output file
	outFile, err := os.Create(*outFl)
	if err != nil {
		fmt.Printf("cannot create output file: %s\n", err)
		os.Exit(1)
	}

	usernames, err := extractUsername(store)
	if err != nil {
		fmt.Printf("cannot extract usernames: %s\n", err)
		os.Exit(1)
	}
	escrows, err := extractEscrow(store)
	if err != nil {
		fmt.Printf("cannot extract escrows: %s\n", err)
		os.Exit(1)
	}
	contracts, err := extractContracts(store)
	if err != nil {
		fmt.Printf("cannot extract contracts: %s\n", err)
		os.Exit(1)
	}

	wallets, err := extractWallets(store)
	if err != nil {
		fmt.Printf("cannot extract wallets: %s\n", err)
		os.Exit(1)
	}

	outJson := Out{
		Username: usernames,
		Escrow:   escrows,
		Contract: contracts,
		Wallets:  wallets,
	}
	enc := json.NewEncoder(outFile)
	enc.SetIndent("", "\t")
	if err := enc.Encode(outJson); err != nil {
		fmt.Printf("cannot write to file: %s\n", err)
		os.Exit(1)
	}
}

func extractUsername(store *app.CommitStore) ([]tokenFormat, error) {
	it := orm.IterAll("tokens")
	var out []tokenFormat
	for {
		var token username.Token
		switch key, err := it.Next(store.CheckStore(), &token); {
		case err == nil:
			owner, err := token.Owner.Bech32String(prefix)
			if err != nil {
				return nil, err
			}
			out = append(out, tokenFormat{
				Username: string(key),
				Targets:  token.Targets,
				Owner:    owner,
			})
		case errors.ErrIteratorDone.Is(err):
			return out, nil
		default:
			return nil, err
		}
	}
}

func extractEscrow(store *app.CommitStore) ([]escrowFormat, error) {
	it := orm.IterAll("esc")
	wb := cash.NewBucket()

	var out []escrowFormat
	for {
		var e escrow.Escrow
		switch key, err := it.Next(store.CheckStore(), &e); {
		case err == nil:
			c, err := wb.Get(store.CheckStore(), key)
			if err != nil {
				return nil, err
			}
			coins := cash.AsCoins(c)

			address, err := e.Address.Bech32String(prefix)
			if err != nil {
				return nil, err
			}
			source, err := e.Source.Bech32String(prefix)
			if err != nil {
				return nil, err
			}
			arbiter, err := e.Arbiter.Bech32String(prefix)
			if err != nil {
				return nil, err
			}
			destination, err := e.Destination.Bech32String(prefix)
			if err != nil {
				return nil, err
			}
			out = append(out, escrowFormat{
				Address:     address,
				Source:      source,
				Arbiter:     arbiter,
				Destination: destination,
				Timeout:     e.Timeout,
				Amount:      coins,
			})
		case errors.ErrIteratorDone.Is(err):
			return out, nil
		default:
			return nil, err
		}
	}
}

func extractContracts(store *app.CommitStore) ([]contractFormat, error) {
	it := orm.IterAll("contracts")
	var out []contractFormat
	for {
		var e multisig.Contract
		switch key, err := it.Next(store.CheckStore(), &e); {
		case err == nil:
			out = append(out, contractFormat{
				Participants:        e.Participants,
				ActivationThreshold: e.ActivationThreshold,
				AdminThreshold:      e.AdminThreshold,
				Address:             key,
			})
		case errors.ErrIteratorDone.Is(err):
			return out, nil
		default:
			return nil, err
		}
	}
}

func extractWallets(store *app.CommitStore) ([]genesisAccount, error) {
	it := orm.IterAll("cash")
	var out []genesisAccount
	for {
		var w cash.Set
		switch key, err := it.Next(store.CheckStore(), &w); {
		case err == nil:
			s := cash.Set{
				Coins: w.Coins,
			}
			k := weave.NewAddress(key)
			if err != nil {
				return nil, err
			}
			address, err := k.Bech32String(prefix)
			if err != nil {
				return nil, err
			}

			out = append(out, genesisAccount{
				Address: address,
				Set:     s,
			})
		case errors.ErrIteratorDone.Is(err):
			return out, nil
		default:
			return nil, err
		}
	}
}
