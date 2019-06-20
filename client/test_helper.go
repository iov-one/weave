package client

import (
	"context"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/abci/types"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/test"
)

// Runner is an interface that would allow us more flexibility in terms of
// types passed to the helper
type Runner interface {
	Run() int
}

// TestWithTendermint provides adaptive startup capabilities and allows
// supplying a callback to initialize test resources dependent on tendermint
// node
func TestWithTendermint(app types.Application, cb func(*nm.Node), m Runner) int {
	n := rpctest.StartTendermint(app)
	cb(n)

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
	_ = n.Stop()
	n.Wait()
	return code
}
