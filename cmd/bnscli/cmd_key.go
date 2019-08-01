package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/crypto/bech32"
	"github.com/stellar/go/exp/crypto/derivation"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ed25519"
)

func cmdKeygen(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Read mnemonic and generate a new private key.

When successful a new file with binary content containing private key is
created. This command fails if the private key file already exists.
`)
		fl.PrintDefaults()
	}
	var (
		keyPathFl = fl.String("key", env("BNSCLI_PRIV_KEY", os.Getenv("HOME")+"/.bnsd.priv.key"),
			"Path to the private key file that transaction should be signed with. You can use BNSCLI_PRIV_KEY environment variable to set it.")
		pathFl = fl.String("path", "m/44'/234'/0'", "Derivation path as described in BIP-44.")
	)
	fl.Parse(args)

	if _, err := os.Stat(*keyPathFl); !os.IsNotExist(err) {
		// Do not allow to overwrite already existing private key. User
		// must manually delete it first to ensure we do not delete
		// such crucial data by an accident (bad command usage).
		return fmt.Errorf("private key file %q already exists, delete this file and try again", *keyPathFl)
	}

	mnemonic, err := readInput(input)
	if err != nil {
		return fmt.Errorf("cannot read mnemonic: %s", err)
	}

	priv, err := keygen(string(mnemonic), *pathFl)
	if err != nil {
		return fmt.Errorf("cannot generate key: %s", err)
	}

	fd, err := os.OpenFile(*keyPathFl, os.O_CREATE|os.O_WRONLY, 0400)
	if err != nil {
		return fmt.Errorf("cannot create public key file: %s", err)
	}
	defer fd.Close()

	if _, err := fd.Write(priv); err != nil {
		return fmt.Errorf("cannot write private key: %s", err)
	}
	if err := fd.Close(); err != nil {
		return fmt.Errorf("cannot close private key file: %s", err)
	}
	return nil
}

// keygen returns a private key generated using given mnemonic and derivation
// path.
func keygen(mnemonic, derivationPath string) (ed25519.PrivateKey, error) {
	if err := validateMnemonic(string(mnemonic)); err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %s", err)
	}

	// We do not allow for passphrase.
	seed := bip39.NewSeed(string(mnemonic), "")

	key, err := derivation.DeriveForPath(derivationPath, seed)
	if err != nil {
		return nil, fmt.Errorf("cannot deriviate master key from seed: %s", err)
	}

	_, priv, err := ed25519.GenerateKey(bytes.NewReader(key.Key))
	if err != nil {
		return nil, fmt.Errorf("cannot generate ed25519 private key: %s", err)
	}
	return priv, nil
}

// isMnemonicValid returns true if given mnemonic string is valid. Whitespaces
// are relevant.
//
// Use this instead of bip39.IsMnemonicValid because this function ensures the
// checksum consistency. bip39.IsMnemonicValid does not test the checksum. It
// also ignores whitespaces.
//
// This function ensures that the mnemonic is a single space separated list of
// words as this is important during seed creation.
func validateMnemonic(mnemonic string) error {
	// A lazy way to check that words are exactly single space separated.
	expected := strings.Join(strings.Fields(mnemonic), " ")
	if mnemonic != expected {
		return errors.New("whitespace violation")
	}

	// Entropy generation does base validation of checking if words are
	// valid and in the right amount. It also tests the checksum.
	if _, err := bip39.EntropyFromMnemonic(mnemonic); err != nil {
		return fmt.Errorf("entropy: %s", err)
	}

	return nil
}

func cmdKeyaddr(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Print out a hex-address associated with your private key.
`)
		fl.PrintDefaults()
	}
	var (
		keyPathFl = fl.String("key", env("BNSCLI_PRIV_KEY", os.Getenv("HOME")+"/.bnsd.priv.key"),
			"Path to the private key file that transaction should be signed with. You can use BNSCLI_PRIV_KEY environment variable to set it.")
		bechPrefixFl = fl.String("bp", "iov", "Bech32 prefix.")
	)
	fl.Parse(args)

	raw, err := ioutil.ReadFile(*keyPathFl)
	if err != nil {
		return fmt.Errorf("cannot read private key file: %s", err)
	}

	if len(raw) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key length: %d", len(raw))
	}

	key := &crypto.PrivateKey{
		Priv: &crypto.PrivateKey_Ed25519{
			Ed25519: raw,
		},
	}

	bech, err := toBech32(*bechPrefixFl, key.PublicKey().GetEd25519())
	if err != nil {
		return fmt.Errorf("cannot generate bech32 address format: %s", err)
	}

	fmt.Fprintf(output, "bech32\t%s\n", bech)
	fmt.Fprintf(output, "hex\t%s\n", key.PublicKey().Address())
	return nil
}

// toBech32 computes the bech32 address representation as described in
// https://github.com/iov-one/iov-core/blob/8846fed17443766a9ad9c908c3d7fc9d205e02ef/docs/address-derivation-v1.md#deriving-addresses-from-keypairs
func toBech32(prefix string, pubkey []byte) ([]byte, error) {
	data := append([]byte("sigs/ed25519/"), pubkey...)
	hash := sha256.Sum256(data)
	bech, err := bech32.Encode(prefix, hash[:20])
	if err != nil {
		return nil, fmt.Errorf("cannot compute bech32: %s", err)
	}
	return bech, nil
}

func cmdMnemonic(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Generate and print out a mnemonic. Keep the result in safe place!
`)
		fl.PrintDefaults()
	}
	var (
		bitSizeFl = fl.Uint("size", 256, "Bit size of the entropy. Must be between 128 and 256.")
	)
	fl.Parse(args)

	entropy, err := bip39.NewEntropy(int(*bitSizeFl))
	if err != nil {
		return fmt.Errorf("cannot create entropy instance: %s", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return fmt.Errorf("cannot create mnemonic instance: %s", err)
	}

	_, err = fmt.Fprintln(output, mnemonic)
	return err
}
