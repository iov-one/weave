# Blockchain name service - bns


## Remote acceptance tests

To execute the test scenarios against a testnet pass the address and a delay to not hit rate limits
```bash
go test -v  ./cmd/bnsd/scenarios/...  -address=https://<testnet-domain>:443 -delay=500ms
```

## Running a local instance

This app relies on a separate tendermint process to drive it. It is helpful to first read a primer on
[tendermint](https://tendermint.readthedocs.io/en/master/introduction.html) as well as the documentation on the [tendermint CLI](https://tendermint.readthedocs.io/en/master/using-tendermint.html).


### Dependencies

- [tendermint](https://github.com/tendermint/tendermint)
- [weave](https://github.com/iov-one/weave)

Versions of both are pinned down in the [weave respository](https://github.com/iov-one/weave/blob/master/Gopkg.lock).

### Running manually

In order to run a node, its state must be first initialized. this is done by running `init` commands.

Make sure to set `TM_VERSION` to the right tendermint version (ie. 0.27.4).
You can change `BNS_HOME` to any directory. This is where the application state is saved.

```sh
$ export TM_VERSION='x.xx.x'
$ export BNS_HOME="$HOME/bns_home"
$ mkdir -p $BNS_HOME
$ docker run \
    -v $BNS_HOME:/tmhome \
    -it \
    --rm \
    iov1/tendermint:$TM_VERSION init \
        --home /tmhome
$ docker run \
    --rm \
    -it \
    -v $BNS_HOME:/bnshome \
    iov1/bnsd:latest \
        -home=/bnshome init
```


Once the state is initialized, `bnsd` instance can be started.

```sh
$ docker run \
    --rm \
    -it \
    -v $BNS_HOME:/bnshome \
    iov1/bnsd:latest \
        -home=/bnshome start \
        -bind=unix:///bnshome/app.sock
$ docker run -v $BNS_HOME:/tmhome \
    -p 26656:26656 \
    -p 26657:26657 \
    -it \
    --rm \
    iov1/tendermint:$TM_VERSION node \
        --home /tmhome \
        --proxy_app="unix:///tmhome/app.sock" \
        --moniker="local"
```
