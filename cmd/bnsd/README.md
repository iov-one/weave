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

Make sure to set `TM_VERSION` to the right tendermint version (ie. 0.27.4)

```
export TM_VERSION='x.xx.x'
docker volume create bns-volume
docker run -v bns-volume:/tmhome  -it --rm iov1/tendermint:$TM_VERSION init --home /tmhome
docker run --rm -it -v bcp-volume:/bcphome iov1/bcpd:latest -home=/bcphome init
```


Once the state is initialized, `bpcd` instance can be started.

```
docker run --rm -it -v bcp-volume:/bcphome iov1/bcpd:latest -home=/bcphome start -bind=unix:///bcphome/app.sock
docker run -v bcp-volume:/tmhome -p 46656:46656 -p 46657:46657  -it --rm iov1/tendermint:$TM_VERSION node --home /tmhome --proxy_app="unix:///tmhome/app.sock"
```
