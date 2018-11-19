package client

import (
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/client"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

/***
These are some helper functions to make a connection
to a full node.

Right now, they can call either tendermint/rpc/client
or a local implementation that overrides this,
as we need to make changes
***/

// NewLocalConnection wraps an in-process node with a client, useful for tests
func NewLocalConnection(node *nm.Node) client.Client {
	// This uses the standard tendermint implementation
	return client.NewLocal(node)
}

// NewHTTPConnection takes a URL and sends all requests to the remote node
func NewHTTPConnection(remote string) client.Client {
	// This uses a custom implementation with support for https/wss
	// We can make local changes easily in tools, and add them back
	// upstream as we just copied some classes over.
	return rpcclient.NewHTTP(remote, "/websocket")
}
