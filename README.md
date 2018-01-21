# Confio Weave

Confio weave is a framework for quickly building your custom
ABCI app to run on a tendermint-based blockchain. It provides
much commonly used functionality that can quickly be imported
in your custom chain, as well as a simple framework for adding
the custom functionality unique to your project.

**Note: Requires Go 1.9+**

It is inspired by the routing and middleware model of many web
application frameworks, and informed by years of wrestling with
blockchain state machines. More directly, it is based on the
offical cosmos-sdk (both the
[0.8 release](https://github.com/cosmos/cosmos-sdk/tree/v0.8.0) as well as the
future [0.9 release aka sdk2](https://github.com/cosmos/cosmos-sdk/tree/sdk2)).

While both of those are extremely powerful and flexible and contain
advanced features, there have a steep learning curve for novice users.
Thus, this library aims to favor simplicity over power when there is
a choice. If you hit limitations in the design of this library
(such as maintaining multiple merkle stores in one app), I highly
advise you to use
[the official cosmos sdk](https://github.com/cosmos/cosmos-sdk).
On the other hand, if you want to try out tendermint, or have a
design that doesn't require an advanced setup, you should try this
library and give feedback, especially on ease-of-use.

In addition to building the ABCI app itself, this contains a framework
to quickly build a CLI tool that uses light-client proofs to interact
with the blockchain as quick scaffolding to test out your application.

For more details on the design goals, see the [Design Document]()

## Prerequisites

* [golang 1.9+](https://golang.org/doc/install)


## Instructions

TODO: vendor, test, install (examples), import
