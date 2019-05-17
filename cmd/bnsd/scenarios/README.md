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


All test should call `bnsdtest.StartBnsd` to create an independent bnsd
instance. Many commonly used values are returned as part of the `EnvConf`.

Returned `env` contains `Client` attribute. It is configured to connect to the
right `bnsd` node. Use it to interact with the network, for example to sign or
broadcast a transaction.

To acquire a transaction sequence, use `client` nonce functionality.

```go
env, cleanup := bnsdtest.StartBnsd(t)
defer cleanup()

aliceNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())
seq, err := aliceNonce.Next()
```

If you need coins, transfer them from the `env.Alice` account together with
`bnsdtest.SeedAccountWithTokens` function.

> Rule of thumb is to do smoke tests. If a message is not rejected there is no
> need to check all the details as we have unit test coverage for that in
> place.
