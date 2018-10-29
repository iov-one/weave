package rpcclient

import (
	"os"
	"testing"

	"github.com/tendermint/abci/example/kvstore"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"
	cmn "github.com/tendermint/tmlibs/common"
)

var node *nm.Node

func TestMain(m *testing.M) {
	// start a tendermint node (and kvstore) in the background to test against
	app := kvstore.NewKVStoreApplication()
	node = rpctest.StartTendermint(app)

	// make sure two blocks are made before any tests start
	client.WaitForHeight(client.NewLocal(node), 2, nil)

	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}

// MakeTxKV returns a text transaction, allong with expected key, value pair
func MakeTxKV() ([]byte, []byte, []byte) {
	k := []byte(cmn.RandStr(8))
	v := []byte(cmn.RandStr(8))
	return k, v, append(k, append([]byte("="), v...)...)
}
