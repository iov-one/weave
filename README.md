# bov

Blockchain of value reference implementation

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
tendermint init --home $HOME/.bov
bov init  # adds app-specific options

# run the app
tendermint node --home $HOME/.bov > /tmp/tendermint.log &
bov start
```

Note that this app relies on a separate tendermint process
to drive it. It is helpful to first read a primer on
[tendermint](https://tendermint.readthedocs.io/en/master/introduction.html)
as well as the documentation on the
[tendermint cli commands](https://tendermint.readthedocs.io/en/master/using-tendermint.html).
