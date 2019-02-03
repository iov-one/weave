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

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/validators"
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
	// Cut out command from the parameters, so that the flag module parse correctly.
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	if err := cmd(); err != nil {
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
var commands = map[string]func() error{
	"list": cmdList,
	"add":  cmdAdd,
}

func cmdList() error {
	var (
		tmAddrFl = flag.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
	)
	flag.Parse()

	info, err := listValidators(*tmAddrFl)
	if err != nil {
		return fmt.Errorf("cannot list validators: %s\n", err)
	}
	b, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot serialize: %s", err)
	}
	os.Stdout.Write(b)
	return nil
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

func cmdAdd() error {
	var (
		tmAddrFl = flag.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
		pubKeyFl = flag.String("pubkey", "", "Base64 encoded, ed25519 public key.")
		hexKeyFl = flag.String("key", "", "Hex encoded, private key of the validator that is to be added/updated.")
		powerFl  = flag.Int64("power", 10, "Validator node power. Set to 0 to delete a node.")
	)
	flag.Parse()

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

	if *pubKeyFl == "" {
		return errors.New("private key is required")
	}
	key, err := decodePrivateKey(*hexKeyFl)
	if err != nil {
		return fmt.Errorf("cannot decode private key: %s", err)
	}

	addValidatorTx := client.SetValidatorTx(
		&validators.ValidatorUpdate{
			Pubkey: validators.Pubkey{
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
