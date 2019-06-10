package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/sigs"
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
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
		keyPathFl = fl.String("key", env("BNSCLI_PRIV_KEY", os.Getenv("HOME")+"/.bnsd.priv.key"),
			"Path to the private key file that transaction should be signed with. You can use BNSCLI_PRIV_KEY environment variable to set it.")
	)
	fl.Parse(args)

	if *keyPathFl == "" {
		return errors.New("private key is required")
	}
	key, err := decodePrivateKey(*keyPathFl)
	if err != nil {
		return fmt.Errorf("cannot load private key: %s", err)
	}

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
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
		sig, err := sigs.SignTx(key, tx, genesis.ChainID, seq)
		if err != nil {
			return fmt.Errorf("cannot sign transaction: %s", err)
		}
		tx.Signatures = append(tx.Signatures, sig)
	}

	_, err = writeTx(output, tx)
	return err
}

func decodePrivateKey(filepath string) (*crypto.PrivateKey, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q file: %s", filepath, err)
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
