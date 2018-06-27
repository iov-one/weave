package utils

import (
	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/confio/weave"
	"github.com/confio/weave/app"
	"github.com/confio/weave/x/sigs"
	"github.com/iov-one/bcp-demo/x/namecoin"
)

// BcpClient is a tendermint client wrapped to provide
// simple access to the data structures used in bcp-demo
type BcpClient struct {
	conn client.Client
}

// NewClient wraps a BcpClient around an existing
// tendermint client connection.
func NewClient(conn client.Client) *BcpClient {
	return &BcpClient{
		conn: conn,
	}
}

type Nonce struct {
	client    *BcpClient
	addr      weave.Address
	nonce     int64
	fromQuery bool
}

func NewNonce(client *BcpClient, addr weave.Address) *Nonce {
	return &Nonce{client: client, addr: addr}
}

// Query always queries the blockchain for the next nonce
func (n *Nonce) Query() (int64, error) {
	user, err := n.client.GetUser(n.addr)
	if err != nil {
		return 0, err
	}
	if user != nil {
		n.nonce = user.UserData.Sequence
	} else {
		n.nonce = 0 // new account starts at 0
	}
	n.fromQuery = true
	return n.nonce, nil
}

// Next will use a cached value if present, otherwise Query
// It will always increment by 1, assuming last nonce
// was properly used. This is designed for cases where
// you want to rapidly generate many tranasactions without
// querying the blockchain each time
func (n *Nonce) Next() (int64, error) {
	if !n.fromQuery && n.nonce == 0 {
		return n.Query()
	}
	n.nonce++
	n.fromQuery = false
	return n.nonce, nil
}

//************ generic (weave) functionality *************//

// AbciResponse contains a query result:
// a (possibly empty) list of key-value pairs, and the height
// at which it queried
type AbciResponse struct {
	// a list of key/value pairs
	Models []weave.Model
	Height int64
}

// AbciQuery calls abci query on tendermint rpc,
// verifies if it is an error or empty, and if there is
// data pulls out the ResultSets from keys and values into
// a useful AbciResponse struct
func (b *BcpClient) AbciQuery(path string, data []byte) (AbciResponse, error) {
	var out AbciResponse

	q, err := b.conn.ABCIQuery(path, data)
	if err != nil {
		return out, err
	}
	resp := q.Response
	if resp.IsErr() {
		return out, errors.Errorf("(%d): %s", resp.Code, resp.Log)
	}
	out.Height = resp.Height

	if len(resp.Key) == 0 {
		return out, nil
	}

	// assume there is data, parse the result sets
	var keys, vals app.ResultSet
	err = keys.Unmarshal(resp.Key)
	if err != nil {
		return out, err
	}
	err = vals.Unmarshal(resp.Value)
	if err != nil {
		return out, err
	}

	out.Models, err = app.JoinResults(&keys, &vals)
	return out, err
}

// BroadcastTxResponse is the result of submitting a transaction
type BroadcastTxResponse struct {
	Error    error                           // not-nil if there was an error sending
	Response *ctypes.ResultBroadcastTxCommit // not-nil if we got response from node
}

// IsError returns the error for failure if it failed,
// or null if it succeeded
func (b BroadcastTxResponse) IsError() error {
	if b.Error != nil {
		return b.Error
	}
	if b.Response.CheckTx.IsErr() {
		ctx := b.Response.CheckTx
		return errors.Errorf("CheckTx error: (%d) %s", ctx.Code, ctx.Log)
	}
	if b.Response.DeliverTx.IsErr() {
		dtx := b.Response.DeliverTx
		return errors.Errorf("CheckTx error: (%d) %s", dtx.Code, dtx.Log)
	}
	return nil
}

// BroadcastTx serializes a signed transaction and writes to the
// blockchain. It returns when the tx is committed to the
// blockchain.
//
// If you want high-performance, parallel sending, use BroadcastTxAsync
func (b *BcpClient) BroadcastTx(tx weave.Tx) BroadcastTxResponse {
	out := make(chan BroadcastTxResponse, 1)
	go b.BroadcastTxAsync(tx, out)
	res := <-out
	return res
}

// BroadcastTxAsync can be run in a goroutine and will output
// the result or error to the given channel.
// Useful if you want to send many tx in parallel
func (b *BcpClient) BroadcastTxAsync(tx weave.Tx, out chan<- BroadcastTxResponse) {
	defer close(out)

	data, err := tx.Marshal()
	if err != nil {
		out <- BroadcastTxResponse{Error: err}
		return
	}

	// TODO: make this async, maybe adjust return value
	res, err := b.conn.BroadcastTxCommit(data)
	msg := BroadcastTxResponse{
		Error:    err,
		Response: res,
	}
	out <- msg
}

//************* app-specific data structures **********//

// WalletResponse is a response on a query for a wallet
type WalletResponse struct {
	Address weave.Address
	Wallet  namecoin.Wallet
	Height  int64
}

// GetWallet will return a wallet given an address
// If non wallet is present, it will return (nil, nil)
// Error codes are used when the query failed on the server
func (b *BcpClient) GetWallet(addr weave.Address) (*WalletResponse, error) {
	// make sure we send a valid address to the server
	err := addr.Validate()
	if err != nil {
		return nil, errors.WithMessage(err, "Invalid Address")
	}

	resp, err := b.AbciQuery("/wallets", addr)
	if err != nil {
		return nil, err
	}
	if len(resp.Models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.Models[0]

	// make sure the return value is expected
	acct := walletKeyToAddr(model.Key)
	if !addr.Equals(acct) {
		return nil, errors.Errorf("Mismatch. Queried %s, returned %s", addr, acct)
	}
	out := WalletResponse{
		Address: acct,
		Height:  resp.Height,
	}

	// parse the value as wallet bytes
	err = out.Wallet.Unmarshal(model.Value)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWalletByName will return a wallet given the wallet name
func (b *BcpClient) GetWalletByName(name string) (*WalletResponse, error) {
	// query by secondary index on name
	resp, err := b.AbciQuery("/wallets/name", []byte(name))
	if err != nil {
		return nil, err
	}
	if len(resp.Models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.Models[0]

	// make sure the return value is expected
	acct := walletKeyToAddr(model.Key)
	err = acct.Validate()
	if err != nil {
		return nil, errors.WithMessage(err, "Returned invalid Address")
	}
	out := WalletResponse{
		Address: acct,
		Height:  resp.Height,
	}

	// TODO: double parse this into result set???

	// parse the value as wallet bytes
	err = out.Wallet.Unmarshal(model.Value)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// key is the address prefixed with "wallet:"
func walletKeyToAddr(key []byte) weave.Address {
	return key[5:]
}

// UserResponse is a response on a query for a User
type UserResponse struct {
	Address  weave.Address
	UserData sigs.UserData
	Height   int64
}

// GetUser will return nonce and public key registered
// for a given address if it was ever used.
// If it returns (nil, nil), then this address never signed
// a transaction before (and can use nonce = 0)
func (b *BcpClient) GetUser(addr weave.Address) (*UserResponse, error) {
	// make sure we send a valid address to the server
	err := addr.Validate()
	if err != nil {
		return nil, errors.WithMessage(err, "Invalid Address")
	}

	resp, err := b.AbciQuery("/auth", addr)
	if err != nil {
		return nil, err
	}
	if len(resp.Models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.Models[0]

	// make sure the return value is expected
	acct := userKeyToAddr(model.Key)
	if !addr.Equals(acct) {
		return nil, errors.Errorf("Mismatch. Queried %s, returned %s", addr, acct)
	}
	out := UserResponse{
		Address: acct,
		Height:  resp.Height,
	}

	// parse the value as wallet bytes
	err = out.UserData.Unmarshal(model.Value)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// key is the address prefixed with "sigs:"
func userKeyToAddr(key []byte) weave.Address {
	return key[5:]
}
