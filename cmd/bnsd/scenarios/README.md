# Acceptance test scenarios

This package contains a set of [go tests](https://golang.org/pkg/testing/) that
ensure happy paths of each functionality is working. Tests are done using [the
black-box technique](https://en.wikipedia.org/wiki/Black-box_testing).


## Run a single test locally

Running a test locally does not require connection to a remote network. Instead
a local `bnsd` instance is started and available throughout all tests.

```sh
go test github.com/iov-one/weave/cmd/bnsd/scenarios \
    -run YOUR_TEST_NAME
```

## Run a single test on a remote network

Running tests on a remote network is using a remote `bnsd` node. Because of
configuration differences it may require a custom configuration. For example a
custom `seed` and a `derivation` path.

```sh
go test github.com/iov-one/weave/cmd/bnsd/scenarios \
    -v \
    -count=1 \
    -run YOUR_TEST_NAME \
    -address=https://bns.hugnet.iov.one:443 \
    -seed=752def518b49a7b0584821126ce26b5ffa656f3378c2064924c1526ed6425c8c1081ef6b63732b56cbbb3e38beae3868460b0780684d2a6ad23f5852229c1e68 \
    -derivation="m/4804438'/0'"
```

## Create a new test

Each test must fulfill [go testing framework
requirements](https://golang.org/pkg/testing/#pkg-overview).

All tests are sharing a configuration that is setup in the `main_test.go`. You
can find there definitions of useful flags and variables. When testing using a
local `bnsd` instance a genesis defined in this file is loaded.

In `main_test.go` a `bnsClient` global variable is defined and initialized. It
is configured to connect to the right `bnsd` node. Use it to interact with the
network, for example to sign or broadcast a transaction.

To acquire a transaction sequence, use `client` nonce functionality.

```go
aliceNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
seq, err := aliceNonce.Next()
```

If you need coins, transfer them from the `alice` account.

> Rule of thumb is to do smoke tests. If a message is not rejected there is no
> need to check all the details as we have unit test coverage for that in
> place.
