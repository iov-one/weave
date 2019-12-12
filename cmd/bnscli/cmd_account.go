package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
)

func cmdRegisterDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for registering a domain. Transaction must be signed by an
address defined by the account functionality configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl          = fl.String("domain", "", "Domain name to register.")
		adminFl           = flAddress(fl, "admin", "", "An address that the newly registered domain will belong to. Transaction does not have to be signed by this address.")
		hasSuperuserFl    = fl.Bool("superuser", true, "Domain has a superuser account?")
		thirdPartyTokenFl = fl.String("third-party-token", "", "Third party token.")
		accountRenewFl    = fl.Duration("account-renew", 30*24*time.Hour, "Account renewal duration.")
	)
	fl.Parse(args)

	msg := account.RegisterDomainMsg{
		Metadata:        &weave.Metadata{Schema: 1},
		Domain:          *domainFl,
		Admin:           *adminFl,
		HasSuperuser:    *hasSuperuserFl,
		ThirdPartyToken: []byte(*thirdPartyTokenFl),
		AccountRenew:    weave.AsUnixDuration(*accountRenewFl),
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountRegisterDomainMsg{
			AccountRegisterDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdWithAccountMsgFee(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Attach a message fee information to given transaction.

This functionality is intended to extend RegisterDomainor
ReplaceAccountMsgFeesMsg messages.
		`)
		fl.PrintDefaults()
	}
	var (
		pathFl   = fl.String("path", "account/register_account_msg", "Message path.")
		amountFl = flCoin(fl, "amount", "1 IOV", "Fee amount.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the transaction: %s", err)
	}

	switch msg := msg.(type) {
	case *account.RegisterDomainMsg:
		msg.MsgFees = append(msg.MsgFees, account.AccountMsgFee{
			MsgPath: *pathFl, Fee: *amountFl,
		})
	case *account.ReplaceAccountMsgFeesMsg:
		msg.NewMsgFees = append(msg.NewMsgFees, account.AccountMsgFee{
			MsgPath: *pathFl, Fee: *amountFl,
		})
	default:
		return fmt.Errorf("unsupported transaction message: %T", msg)
	}

	// Serialize back the transaction from the input. It was modified.
	_, err = writeTx(output, tx)
	return err
}

func cmdRegisterAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for registering an account within a given domain.
		`)
		fl.PrintDefaults()
	}
	var (
		nameFl            = fl.String("name", "", "Account name")
		domainFl          = fl.String("domain", "", "Account domain.")
		adminFl           = flAddress(fl, "owner", "", "An address that the newly registered account will belong to.")
		thirdPartyTokenFl = fl.String("third-party-token", "", "Third party token.")
	)
	fl.Parse(args)

	msg := account.RegisterAccountMsg{
		Metadata:        &weave.Metadata{Schema: 1},
		Name:            *nameFl,
		Domain:          *domainFl,
		Owner:           *adminFl,
		ThirdPartyToken: []byte(*thirdPartyTokenFl),
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountRegisterAccountMsg{
			AccountRegisterAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdWithAccountTarget(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Attach a blockchain address information to given transaction.

This functionality is intended to extend RegisterAccountMsg or
ReplaceAccountTargetsMsg.
		`)
		fl.PrintDefaults()
	}
	var (
		blockchainFl = fl.String("bc", "", "Blockchain network ID.")
		addressFl    = fl.String("address", "", "String representation of the blochain address on this network.")
	)
	fl.Parse(args)

	tx, _, err := readTx(input)
	if err != nil {
		return fmt.Errorf("cannot read input transaction: %s", err)
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return fmt.Errorf("cannot extract message from the transaction: %s", err)
	}

	switch msg := msg.(type) {
	case *account.RegisterAccountMsg:
		msg.Targets = append(msg.Targets, account.BlockchainAddress{
			BlockchainID: *blockchainFl,
			Address:      *addressFl,
		})
	case *account.ReplaceAccountTargetsMsg:
		msg.NewTargets = append(msg.NewTargets, account.BlockchainAddress{
			BlockchainID: *blockchainFl,
			Address:      *addressFl,
		})
	default:
		return fmt.Errorf("unsupported transaction message: %T", msg)
	}

	// Serialize back the transaction from the input. It was modified.
	_, err = writeTx(output, tx)
	return err
}

func cmdUpdateAccountConfiguration(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to change account extension configuration.
		`)
		fl.PrintDefaults()
	}
	var (
		ownerFl             = flAddress(fl, "owner", "", "Address of the owner.")
		validDomainFl       = fl.String("valid-domain", "", "Regular expression defining a rule for a valid domain.")
		validNameFl         = fl.String("valid-name", "", "Regular expression defining a rule for a valid name.")
		validBlockchainID   = fl.String("valid-bl-id", "", "Regular expression defining a rule for a valid blockchain ID string.")
		validBlockchainAddr = fl.String("valid-bl-address", "", "Regular expression defining a rule for a valid blockchain address string.")
		domainRenewFl       = fl.Duration("domain-renew", 0, "Domain renew time.")
	)
	fl.Parse(args)

	msg := account.UpdateConfigurationMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Patch: &account.Configuration{
			Metadata:               &weave.Metadata{Schema: 1},
			Owner:                  *ownerFl,
			ValidDomain:            *validDomainFl,
			ValidName:              *validNameFl,
			ValidBlockchainID:      *validBlockchainID,
			ValidBlockchainAddress: *validBlockchainAddr,
			DomainRenew:            weave.AsUnixDuration(*domainRenewFl),
		},
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountUpdateConfigurationMsg{
			AccountUpdateConfigurationMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTransferDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Transfer a domain by setting a new admin.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain to transfer")
		adminFl  = flAddress(fl, "admin", "", "Address of the new admin.")
	)
	fl.Parse(args)

	msg := account.TransferDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		NewAdmin: *adminFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountTransferDomainMsg{
			AccountTransferDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdTransferAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Transfer an account by setting a new owner.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain that transferred account belongs to.")
		nameFl   = fl.String("name", "", "Name of the account to transferto transfer")
		ownerFl  = flAddress(fl, "owner", "", "Address of the new owner.")
	)
	fl.Parse(args)

	msg := account.TransferAccountMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
		NewOwner: *ownerFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountTransferAccountMsg{
			AccountTransferAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdRenewDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to renew a domain by extending its expiration time.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain.")
	)
	fl.Parse(args)

	msg := account.RenewDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountRenewDomainMsg{
			AccountRenewDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdRenewAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to renew a domain by extending its expiration time.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain that this account belongs to.")
		nameFl   = fl.String("name", "", "Account name")
	)
	fl.Parse(args)

	msg := account.RenewAccountMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountRenewAccountMsg{
			AccountRenewAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdAddAccountCertificate(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to add a certificate to an account.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain that this account belongs to.")
		nameFl   = fl.String("name", "", "Account name")
		certFl   = fl.String("cert-file", "", "Path to a certificate file.")
	)
	fl.Parse(args)

	rawcert, err := ioutil.ReadFile(*certFl)
	if err != nil {
		return fmt.Errorf("cannot read certificate: %s", err)
	}

	msg := account.AddAccountCertificateMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Domain:      *domainFl,
		Name:        *nameFl,
		Certificate: rawcert,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountAddAccountCertificateMsg{
			AccountAddAccountCertificateMsg: &msg,
		},
	}
	_, err = writeTx(output, tx)
	return err
}

func cmdDeleteDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete a domain.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain.")
	)
	fl.Parse(args)

	msg := account.DeleteDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountDeleteDomainMsg{
			AccountDeleteDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdDeleteAccount(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete an account that belongs to a given domain.
		`)
		fl.PrintDefaults()
	}
	var (
		nameFl   = fl.String("name", "", "Account name")
		domainFl = fl.String("domain", "", "Domain that this account belongs to.")
	)
	fl.Parse(args)

	msg := account.DeleteAccountMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountDeleteAccountMsg{
			AccountDeleteAccountMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdFlushDomain(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete all accounts that belong to a given domain.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain")
	)
	fl.Parse(args)

	msg := account.FlushDomainMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountFlushDomainMsg{
			AccountFlushDomainMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdReplaceAccountTrarget(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to replace targets for a given account.

Use another command to configure which blockchain addresses should be set as
new targets.
		`)
		fl.PrintDefaults()
	}
	var (
		nameFl   = fl.String("name", "", "Account name")
		domainFl = fl.String("domain", "", "Domain that this account belongs to.")
	)
	fl.Parse(args)

	msg := account.ReplaceAccountTargetsMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
		Name:     *nameFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountReplaceAccountTargetsMsg{
			AccountReplaceAccountTargetsMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdReplaceAccountMsgFees(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to replace message fees for a given domain.

Use another command to configure which fees should be present in the new set.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain.")
	)
	fl.Parse(args)

	msg := account.ReplaceAccountMsgFeesMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   *domainFl,
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountReplaceAccountMsgFeesMsg{
			AccountReplaceAccountMsgFeesMsg: &msg,
		},
	}
	_, err := writeTx(output, tx)
	return err
}

func cmdDelAccountCertificate(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction to delete a single account certificate.
		`)
		fl.PrintDefaults()
	}
	var (
		domainFl = fl.String("domain", "", "Domain that this account belongs to.")
		nameFl   = fl.String("name", "", "Account name")
		certFl   = fl.String("cert-file", "", "Path to a certificate file.")
	)
	fl.Parse(args)

	rawcert, err := ioutil.ReadFile(*certFl)
	if err != nil {
		return fmt.Errorf("cannot read certificate: %s", err)
	}

	hash := sha256.Sum256(rawcert)

	msg := account.DeleteAccountCertificateMsg{
		Metadata:        &weave.Metadata{Schema: 1},
		Domain:          *domainFl,
		Name:            *nameFl,
		CertificateHash: hash[:],
	}
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("given data produce an invalid message: %s", err)
	}
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_AccountDeleteAccountCertificateMsg{
			AccountDeleteAccountCertificateMsg: &msg,
		},
	}
	_, err = writeTx(output, tx)
	return err
}
