package utils

import (
	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/rpc/client"

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

	resp, err := b.abciQuery("/wallets", addr)
	if err != nil {
		return nil, err
	}
	if len(resp.models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.models[0]

	// make sure the return value is expected
	acct := walletKeyToAddr(model.Key)
	if !addr.Equals(acct) {
		return nil, errors.Errorf("Mismatch. Queried %s, returned %s", addr, acct)
	}
	out := WalletResponse{
		Address: acct,
		Height:  resp.height,
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
	resp, err := b.abciQuery("/wallets/name", []byte(name))
	if err != nil {
		return nil, err
	}
	if len(resp.models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.models[0]

	// make sure the return value is expected
	acct := walletKeyToAddr(model.Key)
	err = acct.Validate()
	if err != nil {
		return nil, errors.WithMessage(err, "Returned invalid Address")
	}
	out := WalletResponse{
		Address: acct,
		Height:  resp.height,
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

	resp, err := b.abciQuery("/auth", addr)
	if err != nil {
		return nil, err
	}
	if len(resp.models) == 0 { // empty list or nil
		return nil, nil // no wallet
	}
	// assume only one result
	model := resp.models[0]

	// make sure the return value is expected
	acct := userKeyToAddr(model.Key)
	if !addr.Equals(acct) {
		return nil, errors.Errorf("Mismatch. Queried %s, returned %s", addr, acct)
	}
	out := UserResponse{
		Address: acct,
		Height:  resp.height,
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

type abciResponse struct {
	// a list of key/value pairs
	models []weave.Model
	height int64
}

func (b *BcpClient) abciQuery(path string, data []byte) (abciResponse, error) {
	var out abciResponse

	q, err := b.conn.ABCIQuery(path, data)
	if err != nil {
		return out, err
	}
	resp := q.Response
	if resp.IsErr() {
		return out, errors.Errorf("(%d): %s", resp.Code, resp.Log)
	}
	out.height = resp.Height

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

	out.models, err = app.JoinResults(&keys, &vals)
	return out, err
}
