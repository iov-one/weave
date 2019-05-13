package server

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/tendermint/iavl"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/types"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	iavlstore "github.com/iov-one/weave/store/iavl"
)

const (
	flagUntilError = "error"
	flagMaxTries   = "max"
)

type retryArgs struct {
	dbPath     string
	blockPath  string
	debug      bool
	untilError bool
	maxTries   int
}

func parseRetryArgs(args []string) (retryArgs, error) {
	if len(args) < 2 {
		return retryArgs{}, errors.Wrap(errors.ErrInput,
			"usage: cmd retry <path to abci.db> <path to block.json> [-debug] [-error] [-max=N]")
	}
	res := retryArgs{
		dbPath:    args[0],
		blockPath: args[1],
	}
	getBlockFlags := flag.NewFlagSet("retry", flag.ExitOnError)
	getBlockFlags.BoolVar(&res.debug, flagDebug, false, "print out debug info")
	getBlockFlags.BoolVar(&res.untilError, flagUntilError, false, "retry multiple times until an error appears")
	getBlockFlags.IntVar(&res.maxTries, flagMaxTries, 10, "maximum number of times to retry if -error is passed")
	err := getBlockFlags.Parse(args[2:])
	return res, err
}

// InlineAppGenerator should be implemented by the app/init.go file
type InlineAppGenerator func(weave.CommitKVStore, log.Logger, bool) abci.Application

type appBuilder func(weave.CommitKVStore) abci.Application

func wrapInlineAppGenerator(gen InlineAppGenerator, logger log.Logger, debug bool) appBuilder {
	return func(kv weave.CommitKVStore) abci.Application {
		return gen(kv, logger, debug)
	}
}

// RetryCmd takes the app state and the last block from the file system
// It verifies that they match, then rolls back one block and re-runs the given block
// It will output the new hash after running.
//
// If -error is passed, then it will try -max times until a different app hash results
func RetryCmd(makeApp InlineAppGenerator, logger log.Logger, home string, args []string) error {
	flags, err := parseRetryArgs(args)
	if err != nil {
		return err
	}

	fmt.Println("--> Loading Block")
	blockJSON, err := ioutil.ReadFile(flags.blockPath)
	if err != nil {
		return err
	}
	var block *types.Block
	err = cdc.UnmarshalJSON(blockJSON, &block)
	if err != nil {
		return err
	}

	fmt.Println("--> Loading Database")
	tree, ver, err := readTree(flags.dbPath, 0)
	if err != nil {
		return errors.Wrap(err, "error reading abci data")
	}

	if ver != block.Header.Height {
		return errors.Wrapf(errors.ErrState,
			"height mismatch - block=%d, abcistore=%d", block.Header.Height, ver)
	}

	builder := wrapInlineAppGenerator(makeApp, logger, flags.debug)
	return retryBlock(builder, tree, block, flags.untilError, flags.maxTries)
}

func readTree(dir string, version int) (*iavl.MutableTree, int64, error) {
	db, err := openDb(dir)
	if err != nil {
		return nil, 0, err
	}
	tree := iavl.NewMutableTree(db, 10000) // cache size 10000
	ver, err := tree.LoadVersion(int64(version))
	if ver == 0 {
		return nil, 0, errors.Wrap(errors.ErrState, "iavl tree is empty")
	}
	return tree, ver, err
}

func retryBlock(builder appBuilder, tree *iavl.MutableTree, block *types.Block, untilError bool, maxTries int) error {
	fmt.Printf("Original Height: %d\n", block.Header.Height)
	fmt.Printf("Original Hash: %X\n", tree.Hash())

	same, err := rerunBlock(builder, tree, block)
	if err != nil {
		return err
	}

	for same && untilError && maxTries > 0 {
		maxTries--
		same, err = rerunBlock(builder, tree, block)
		if err != nil {
			return err
		}
	}

	return nil
}

func rerunBlock(builder appBuilder, tree *iavl.MutableTree, block *types.Block) (bool, error) {
	origHash := tree.Hash()
	backHeight := block.Header.Height - 1

	fmt.Printf("Rollback to height: %d\n", backHeight)
	_, err := tree.LoadVersionForOverwriting(backHeight)
	if err != nil {
		return false, err
	}

	// run this block....
	kv := iavlstore.NewCommitStoreFromTree(tree)
	app := builder(kv)

	fmt.Println("---> Begin Block")
	app.BeginBlock(abci.RequestBeginBlock{Hash: block.Header.Hash(), Header: toAbciHeader(block.Header)})
	for i, tx := range block.Txs {
		fmt.Printf("---> Deliver Tx %d\n", i)
		app.DeliverTx(tx)

	}
	fmt.Println("---> End Block")
	app.EndBlock(abci.RequestEndBlock{Height: block.Header.Height})
	hash := app.Commit().Data
	fmt.Printf("Recomputed Hash: %X\n", hash)

	same := bytes.Equal(origHash, hash)
	return same, nil
}

func toAbciHeader(h types.Header) abci.Header {
	lb := h.LastBlockID
	return abci.Header{
		Version: abci.Version{
			Block: uint64(h.Version.Block),
			App:   uint64(h.Version.App),
		},
		ChainID:  h.ChainID,
		Height:   h.Height,
		Time:     h.Time,
		NumTxs:   h.NumTxs,
		TotalTxs: h.TotalTxs,
		LastBlockId: abci.BlockID{
			Hash: lb.Hash,
			PartsHeader: abci.PartSetHeader{
				Total: int32(lb.PartsHeader.Total),
				Hash:  lb.PartsHeader.Hash,
			},
		},
		LastCommitHash:     h.LastCommitHash,
		DataHash:           h.DataHash,
		ValidatorsHash:     h.ValidatorsHash,
		NextValidatorsHash: h.NextValidatorsHash,
		ConsensusHash:      h.ConsensusHash,
		AppHash:            h.AppHash,
		LastResultsHash:    h.LastResultsHash,
		EvidenceHash:       h.EvidenceHash,
		ProposerAddress:    h.ProposerAddress,
	}
}
