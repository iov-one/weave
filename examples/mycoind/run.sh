#!/bin/bash

# Assumes you have everything set up
# and have initialised the chain

docker rm -f iov-tm || true
docker run --name iov-tm --rm -p26657:26657 -v ~/.mycoind:/tendermint iov1/tendermint:0.21.0 init
docker run --name iov-tm -d -p26657:26657 -v ~/.mycoind:/tendermint iov1/tendermint:0.21.0 node --proxy_app="tcp://host.docker.internal:46658"

mycoind start
