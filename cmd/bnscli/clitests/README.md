# `bnscli` tool tests

This is a set of tests executing shell scripts and ensuring the output is as
expected. Tests expect `bnscli` binary to be present in one of the `$PATH`
directories.

## Examples

This directory largely serves as self-documenting examples of how to create
transactions. These can be seen below:

* [Create Multisig](./attach_multisig_id.test)
* [Create batch of send tx](./batch.test)
* [Governance proposal with batch command](./batch_proposal.test)

**TODO** Is this useful?

## Submitting the transaction

After creating a transaction with any of the previous manners, you need to
prepare it for submission. All of the following commands take a
`-tm=https://bns.NETWORK.iov.one:443` like argument to point to the proper
network, where we want to query fee info, nonce, or submit it.
It is usually easier to just `export BNSCLI_TM_URL=https://bns.NETWORK.iov.one:443`
and then ignore repeating the flag on all these commands.

```console
cat unsigned_tx.bin \
	| bnscli with-fee [-amount="0.1 IOV"] \
	| bnscli sign [-key $keyfile] \
	| bnscli submit
```

`with-fee` will query the proper fee for the given transaction (anti-spam fee plus product fee),
unless you specify a manual amount as override.

`sign` will sign with a private key located in `$HOME/.bnsd.priv.key` unless you specify a different
location. It will calculate the address of that key and query the given chain for the proper nonce
before signing.

`submit` will post the signed transaction to the given chain and wait until it is in a block.
This may take a second or two, but remember, the chain is not blocked at this time, you are just
waiting for the next block to be processes. You can run this in parallel, but not with the same
account, or else you will have issues with out-of-order nonces.


### Running tests

To run the tests you need Go. We are using Go's
[testing](https://golang.org/pkg/testing/) package as the test runner.  Enter
`clitest` directory and run:

    $ go test .


### Adding new test

To add a new test, create a file `<test_name>.test` in this directory. It
should be a [Bourne shell](https://en.wikipedia.org/wiki/Bourne_shell) (not
[bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell))) script. Its stdout
will be captured by the test runner and compared with `<test_name>.test.gold`
file content.

Best is to start your test file with the following lines:

    #!/bin/sh
    set -e


### Creating a golden file

Do not create `xxx.test.gold` files by hand. Instead, run the test runner with
the `-gold` flag to regenerate all of them.

    $ go test -gold .

This will overwrite all golden files with new results. Check the changes using
`git diff` command to make sure the output change is expected.
