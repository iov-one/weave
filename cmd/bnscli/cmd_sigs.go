package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/crypto"
)

func cmdSignTransaction(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Sign given transaction. This is decoding a transaction data from standard
input, adds a signature and writes back to standard output signed transaction
content.

`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
		keyFl    = fl.String("key", "", "Hex encoded, private key that transaction should be signed with.")
	)
	fl.Parse(args)

	if *keyFl == "" {
		return errors.New("private key is required")
	}
	key, err := decodePrivateKey(*keyFl)
	if err != nil {
		return fmt.Errorf("cannot decode private key: %s", err)
	}

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}
	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}

	genesis, err := fetchGenesis(*tmAddrFl)
	if err != nil {
		return fmt.Errorf("cannot fetch genesis: %s", err)
	}

	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))
	aNonce := client.NewNonce(bnsClient, key.PublicKey().Address())
	if seq, err := aNonce.Next(); err != nil {
		return fmt.Errorf("cannot get the next sequence number: %s", err)
	} else {
		client.SignTx(&tx, key, genesis.ChainID, seq)
	}

	if raw, err := tx.Marshal(); err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	} else {
		_, err = output.Write(raw)
	}
	return err
}

func decodePrivateKey(hexSeed string) (*crypto.PrivateKey, error) {
	data, err := hex.DecodeString(hexSeed)
	if err != nil {
		return nil, fmt.Errorf("cannot hex decode: %s", err)
	}
	if len(data) != 64 {
		return nil, errors.New("invalid key length")
	}
	key := &crypto.PrivateKey{
		Priv: &crypto.PrivateKey_Ed25519{Ed25519: data},
	}
	return key, nil
}

func fetchGenesis(serverURL string) (*genesis, error) {
	resp, err := http.Get(serverURL + "/genesis")
	if err != nil {
		return nil, fmt.Errorf("cannot fetch: %s", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Result struct {
			Genesis genesis `json:"genesis"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cannot decode response: %s", err)
	}
	return &payload.Result.Genesis, nil
}

type genesis struct {
	ChainID string `json:"chain_id"`
}
