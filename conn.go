package utils

import (
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/client"
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
	return client.NewLocal(node)
}

// NewHTTPConnection takes a URL and sends all requests to the remote node
func NewHTTPConnection(remote string) client.Client {
	return client.NewHTTP(remote, "/websocket")
}
