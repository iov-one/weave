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
        Tendermint node address. Use proper NETWORK name. You can use TM_ADDR
        environment variable to set it. (default "https://bns.NETWORK.iov.one:443")

## Combine commands using UNIX pipe

Each command provides a small portion of functionality expected by any
workflow. Combine them using UNIX pipes to create a powerful pipelines.

## Cookbook

For example usage of commands as well as pipelines, see
[`clitests/`](clitests/) directory. Files with extension `.test` contains short
code snippets.
