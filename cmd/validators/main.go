package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/crypto"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %q\n\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(2)
	}
	// Cut out program name and the command from the parameters, so that
	// the flag module parse correctly.
	if err := cmd(os.Stdin, os.Stdout, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// printUsage writes this application usage summary.
func printUsage(w io.Writer) {
	var cmds []string
	for name := range commands {
		cmds = append(cmds, name)
	}
	fmt.Fprintf(w, `Usage:
	%s <cmd> [options]

Use <cmd> -h to display help for each command.
Available commands: %s
`, os.Args[0], strings.Join(cmds, ", "))
}

// commands is a global register of all commands provided by this program. Each
// command should use flag package to support options and provide help text.
var commands = map[string]func(input io.Reader, output io.Writer, args []string) error{
	"list":            cmdList,
	"add":             cmdAdd,
	"multisig-new":    cmdMultisigNew,
	"multisig-sign":   cmdMultisigSign,
	"multisig-view":   cmdMultisigView,
	"multisig-submit": cmdMultisigSubmit,
}

func cmdList(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
	)
	fl.Parse(args)

	info, err := listValidators(*tmAddrFl)
	if err != nil {
		return fmt.Errorf("cannot list validators: %s\n", err)
	}
	b, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot serialize: %s", err)
	}
	_, err = output.Write(b)
	return err
}

func listValidators(nodeURL string) ([]*validatorInfo, error) {
	req, err := http.NewRequest("GET", nodeURL+"/validators", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create HTTP request: %s", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do HTTP request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1e5))
		return nil, fmt.Errorf("unexpected response: %d %s", resp.StatusCode, string(b))
	}

	var payload struct {
		Result struct {
			Validators []*validatorInfo
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cannot decode response: %s", err)
	}
	return payload.Result.Validators, nil
}

type validatorInfo struct {
	Address     string     `json:"address"`
	PubKey      pubKeyInfo `json:"pub_key"`
	VotingPower string     `json:"voting_power"`
	Accum       string     `json:"accum"`
}

type pubKeyInfo struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func cmdMultisigNew(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Create a new validator update request with multi signature authentication.
Created request is binary serialized and written to standard output.

Returned request must be signed by other parties before it can be submitted.

`)
		fl.PrintDefaults()
	}
	var (
		pubKeyFl       = fl.String("pubkey", "", "Base64 encoded, ed25519 public key.")
		multisigAddrFl = fl.String("multisig", "", "Address of the multisig contract that this request will authenticate with.")
		powerFl        = fl.Int64("power", 10, "Validator node power. Set to 0 to delete a node.")
	)
	fl.Parse(args)

	if *pubKeyFl == "" {
		return errors.New("public key is required")
	}
	pubkey, err := base64.StdEncoding.DecodeString(*pubKeyFl)
	if err != nil {
		return fmt.Errorf("cannot base64 decode public key: %s", err)
	}

	if *multisigAddrFl == "" {
		return errors.New("multisig address is required")
	}

	addValidatorTx := client.SetValidatorTx(
		weave.ValidatorUpdate{
			PubKey: weave.PubKey{
				Type: "ed25519",
				Data: pubkey,
			},
			Power: *powerFl,
		},
	)

	raw, err := addValidatorTx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(raw)
	return err

}

func cmdMultisigView(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Decode and display transaction summary. This command is helpful when reciving a
binary representation of a transaction. Before signing you should check what
kind of operation are you authorizing.
`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}

	var tx bnsd.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}

	// Protobuf compiler is exposing all attributes as JSON as
	// well. This will produce a beautiful summary.
	pretty, err := json.MarshalIndent(tx, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize: %s", err)
	}
	_, err = output.Write(pretty)
	return err
}

func cmdMultisigSign(
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
	var tx bnsd.Tx
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

func cmdMultisigSubmit(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Read binary serialized validator change transaction from standard input and
submit it.

This command is intended to be used to submit multisig authenticated requests.
Make sure to collect enough signatures before submitting.

`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
	)
	fl.Parse(args)

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction from input: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}
	var tx bnsd.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}
	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))

	if err := bnsClient.BroadcastTx(&tx).IsError(); err != nil {
		return fmt.Errorf("cannot broadcast transaction: %s", err)
	}
	return nil

}

func cmdAdd(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
		pubKeyFl = fl.String("pubkey", "", "Base64 encoded, ed25519 public key.")
		hexKeyFl = fl.String("key", "", "Hex encoded, private key of the validator that is to be added/updated.")
		powerFl  = fl.Int64("power", 10, "Validator node power. Set to 0 to delete a node.")
	)
	fl.Parse(args)

	if *pubKeyFl == "" {
		return errors.New("public key is required")
	}
	pubkey, err := base64.StdEncoding.DecodeString(*pubKeyFl)
	if err != nil {
		return fmt.Errorf("cannot base64 decode public key: %s", err)
	}

	genesis, err := fetchGenesis(*tmAddrFl)
	if err != nil {
		return fmt.Errorf("cannot fetch genesis: %s", err)
	}

	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))

	if *hexKeyFl == "" {
		return errors.New("private key is required")
	}
	key, err := decodePrivateKey(*hexKeyFl)
	if err != nil {
		return fmt.Errorf("cannot decode private key: %s", err)
	}

	addValidatorTx := client.SetValidatorTx(
		weave.ValidatorUpdate{
			PubKey: weave.PubKey{
				Type: "ed25519",
				Data: pubkey,
			},
			Power: *powerFl,
		},
	)

	aNonce := client.NewNonce(bnsClient, key.PublicKey().Address())
	if seq, err := aNonce.Next(); err != nil {
		return fmt.Errorf("cannot get the next sequence number: %s", err)
	} else {
		client.SignTx(addValidatorTx, key, genesis.ChainID, seq)
	}
	if err := bnsClient.BroadcastTx(addValidatorTx).IsError(); err != nil {
		return fmt.Errorf("cannot broadcast transaction: %s", err)
	}
	return nil

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
