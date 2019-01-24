# BOV - Blockchain of value reference implementation
This application illustrates integration of application logic with a blockchain solution, in this case [tendermint](https://tendermint.readthedocs.io/en/master/introduction.html).

Currently we only support `latest` tag

Note that this app relies on a separate tendermint process
to drive it. It is helpful to first read a primer on
[tendermint](https://tendermint.readthedocs.io/en/master/introduction.html)
as well as the documentation on the
[tendermint cli commands](https://tendermint.readthedocs.io/en/master/using-tendermint.html).

Maintained by: [IOV One](https://www.iov.one/)

## Dependencies
Tendermint v0.21.0

## Running manually
### Init
```
docker volume create bov-volume
docker run -v bov-volume:/tmhome  -it --rm iov1/tendermint:0.21.0 init --home /tmhome
docker run --rm -it -v bov-volume:/bovhome iov1/bcpd:latest -home=/bovhome init
```
### Run interactively
```
docker run --rm -it -v bov-volume:/bovhome iov1/bcpd:latest -home=/bovhome start -bind=unix:///bovhome/app.sock
docker run -v bov-volume:/tmhome -p 46656:46656 -p 46657:46657  -it --rm iov1/tendermint:0.21.0 node --home /tmhome --proxy_app="unix:///tmhome/app.sock"
```

### In order to connect to this
Consider using [iov-core](https://github.com/iov-one/iov-core)

