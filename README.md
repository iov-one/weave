# bov

Blockchain of value reference implementation

## Prerequisites

* [golang 1.9+](https://golang.org/doc/install)

## Installation

If you have vagrant installed, the simplest way to get going is:

```
vagrant up
vagrant ssh -c "sudo service bov start && sudo service tendermint start"
curl localhost:46657/status
```

`vagrant up` compiles tendermint and `bov-core` from source, sets up
a default init files. This can be modified to use pre-compiled
binaries when we have some stable releases. `tendermint` and `bov`
are both registered as systemd services, and will restart on the
next reboot.

Vagrant exposes the tendermint rpc endpoint (:46657) on the
host machine, so you can point the `bov_server` to that endpoint,
and not worry about the rest of the golang mess.

If you want a custom init file (and who doesn't?), before starting
tendermint, run `vagrant ssh` and edit `.bov/config/genesis.json`.
If you already started it, you must:

```
sudo service tendermint stop
sudo service bov stop
rm -rf .bov/data
```

so you can restart with a new genesis file (tendermint doesn't
like to change the genesis on a running chain, wonder why?)

### Deployment

Vagrant is great for local development, but we actually split out
the logic into 4 shell scripts, so you can use those to deploy to
any VM you wish. Take a look at the Vagrantfile, then use the
scripts in the `deploy` directory. They may need a few tweaks, if
so, feel free to add a PR.

### Dev mode

If you have golang locally and want to modify the code:

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
