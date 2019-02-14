package server

import (
	"flag"
	"fmt"
	"strings"

	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	flagHeight = "height"
)

var cdc = amino.NewCodec()

func init() {
	ctypes.RegisterAmino(cdc)
}

func parseGetBlockArgs(args []string) (string, int64, error) {
	if len(args) == 0 {
		return "", 0, fmt.Errorf("Usage: cmd getblock <path to blockstore.db> [-height=H]")
	}
	var height int
	getBlockFlags := flag.NewFlagSet("getblock", flag.ExitOnError)
	getBlockFlags.IntVar(&height, flagHeight, 0, "height of the block to extract (default latest)")
	err := getBlockFlags.Parse(args[1:])
	return args[0], int64(height), err
}

// GetBlockCmd extracts a block from a blockstore.db and outputs as json
// It takes the last block unless -height is explicitly specified
// It writes the json to stdout
func GetBlockCmd(logger log.Logger, home string, args []string) error {
	dbPath, height, err := parseGetBlockArgs(args)
	if err != nil {
		return err
	}
	db, err := openDb(dbPath)
	if err != nil {
		return err
	}
	store := blockchain.NewBlockStore(db)
	if height == 0 {
		height = store.Height()
	}
	return printBlock(store, height)
}

func openDb(dir string) (dbm.DB, error) {
	if strings.HasSuffix(dir, ".db") {
		dir = dir[:len(dir)-3]
	} else if strings.HasSuffix(dir, ".db/") {
		dir = dir[:len(dir)-4]
	} else {
		return nil, fmt.Errorf("Database directory must end with .db")
	}
	// TODO: doesn't work on windows!
	cut := strings.LastIndex(dir, "/")
	if cut == -1 {
		return nil, fmt.Errorf("Cannot cut paths on %s", dir)
	}
	name := dir[cut+1:]
	db, err := dbm.NewGoLevelDB(name, dir[:cut])
	if err != nil {
		return nil, err
	}
	return db, nil
}

func printBlock(store *blockchain.BlockStore, height int64) error {
	block := store.LoadBlock(height)
	if block == nil {
		return fmt.Errorf("No block for height: %d", height)
	}
	js, err := cdc.MarshalJSONIndent(block, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(js))
	return nil
}
