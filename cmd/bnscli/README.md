# `bnscli`

`bnscli` provides a set of commands to create, modify and submit messages. Each
command provides a minimal set functionality. Commands are intended to be
combined into pipelines using UNIX pipes.

## Command information

To get the list of all available command, type `bnscli`. Each command provides
further explanation. Use `-help` flag to learn more about each command.

    $ bnscli submit -help
    Read binary serialized transaction from standard input and submit it.

    Make sure to collect enough signatures before submitting the transaction.
    -tm string
        Tendermint node address. Use proper NETWORK name. You can use
        BNSCLI_TM_ADDR environment variable to set it. (default
        "https://bns.NETWORK.iov.one:443")

## Combine commands using UNIX pipe

Each command provides a small portion of functionality expected by any
workflow. Combine them using UNIX pipes to create a powerful pipelines.

## Cookbook

For example usage of commands as well as pipelines, see
[`clitests/`](clitests/) directory. Files with extension `.test` contains short
code snippets.

Notice that all pipelines end with `bnscli view`. This displays the
transaction. To submit it, after the transaction is prepared, sign it and
submit it. This is done by piping the transaction to `bnscli`:

```
<build tx with bnscli> | bnscli sign | bnscli submit
```

To sign and submit you must provide the signature key and set tendermint
adderess. Both can be set via environment variables `BNSCLI_PRIV_KEY` and
`BNSCLI_TM_ADDR`.

- [Send funds from the `src` to the `dst` account](clitests/send_tokens.test).
  For example, transfer funds from guarantee to reward account.
- [Add a single or multiple validators](clitests/set_validators.test).
- [Update an electorate](clitests/gov_update-electorate.test) via proposal. For
  example to add a new member to the tech committee.
- [Update configuration of a election
  rule](clitests/gov_update-election-rule.test) via proposal. For example,
  create a proposal to change the quorum for the economic committee.
