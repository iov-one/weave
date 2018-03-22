# Confio Weave
[![Build Status circleCI](https://circleci.com/gh/confio/weave.png?style=shield&circle-token=4a0ef9d48e00e82022cc6750ba63b2fd6d75343c)](https://circleci.com/gh/confio/weave)
[![codecov](https://codecov.io/gh/confio/weave/branch/master/graph/badge.svg)](https://codecov.io/gh/confio/weave/branch/master)
[![LoC](https://tokei.rs/b1/github/confio/weave)](https://github.com/confio/weave)
[![Go Report Card](https://goreportcard.com/badge/github.com/confio/weave)](https://goreportcard.com/report/github.com/confio/weave)
[![API Reference](https://godoc.org/github.com/confio/weave?status.svg
)](https://godoc.org/github.com/confio/weave)
[![license](https://img.shields.io/github/license/confio/weave.svg)](https://github.com/confio/weave/blob/master/LICENSE)

Confio Weave is a framework for quickly building your custom
[ABCI application](https://github.com/tendermint/abci)
to run a blockchain on top of the best-of-class
BFT Proof-of-stake [Tendermint consensus engine](https://tendermint.com).
It provides much commonly used functionality that can
quickly be imported in your custom chain, as well as a
simple framework for adding the custom functionality unique
to your project.

**Note: Requires Go 1.9+**

It is inspired by the routing and middleware model of many web
application frameworks, and informed by years of wrestling with
blockchain state machines. More directly, it is based on the
offical cosmos-sdk, both the
[0.8 release](https://github.com/cosmos/cosmos-sdk/tree/v0.8.0) as well as the
future [0.9 rewrite](https://github.com/cosmos/cosmos-sdk/tree/develop). Naturally, as I was the main author of 0.8.

While both of those are extremely powerful and flexible
and contain advanced features, they have a steep learning
curve for novice users. Thus, this library aims to favor
simplicity over power when there is a choice. If you hit
limitations in the design of this library (such as
maintaining multiple merkle stores in one app), I highly
advise you to use
[the official cosmos sdk](https://github.com/cosmos/cosmos-sdk).

On the other hand, if you want to try out tendermint, or have a
design that doesn't require an advanced setup, you should try
this library and give feedback, especially on ease-of-use.
The end goal is to make blockchain development almost as
productive as web development (in golang), by providing
defaults and best practices for many choices, while allowing
extreme flexibility in business logic and data modelling.

For more details on the design goals, see the
[Design Document](./docs/design.rst)

## Prerequisites

* [golang 1.9+](https://golang.org/doc/install)


## Instructions

How to build and run a simple app:

```
# this installs all vendor dependencies, as well as the
# tendermint binary
make deps
make install

# initialize an app
tendermint init --home $HOME/.mycoind
mycoind init  # adds app-specific options

# run the app
tendermint node --home $HOME/.mycoind > /tmp/tendermint.log &
mycoind start
```

Note that this app relies on a separate tendermint process
to drive it. It is helpful to first read a primer on
[tendermint](https://tendermint.readthedocs.io/en/master/introduction.html)
as well as the documentation on the
[tendermint cli commands](https://tendermint.readthedocs.io/en/master/using-tendermint.html).
