package client

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tendermint/tendermint/abci/types"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/test"
)

// TestWithTendermint provides adaptive startup capabilities and allows
// supplying a callback to initialize test resources dependent on tendermint
// node
func TestWithTendermint(app types.Application, f func(*nm.Node), m *testing.M) {
	n := rpctest.StartTendermint(app)
	f(n)

	fmt.Println("Wait for first block...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	h, err := NewLocalClient(n).WaitForNextBlock(ctx)
	fmt.Printf("Starting tests with block %d\n", h.Height)

	// Run tests if tendermint started properly
	var code int
	if err == nil {
		code = m.Run()
	} else {
		fmt.Printf("Failed to start tendermint: %s\n", err)
		code = 1
	}

	// and shut down proper at the end
	n.Stop()
	n.Wait()
	os.Exit(code)
}
