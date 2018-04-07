#!/bin/bash

# Assumes you have everything set up
# and have initialised the chain

tendermint node --home ~/.mycoind --p2p.skip_upnp > ~/.mycoind/tendermint.log &
mycoind start