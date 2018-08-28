----------
Validators
----------

Since Tendermint uses a traditional BFT algorithm to reach
consensus on blocks, signatures from specified validator keys
replace hashes used to mine blocks in typical PoW chains.
This also means that the selection of validators is an extremely
important part of the blockchain security, and every validator
should have strong security in place to avoid their private keys
being copied or stolen.

Static Validators
-----------------

In the simplest setup, every node can generate a private key with
``tendermint init``. Note that this is stored as a clear-text file
on the harddrive, so the machine should be well locked-down,
and file permissions double-checked. This file not only contains
the private key itself, but also information on the last block
proposal signed, to avoid double-signing blocks, even in the even of
a restart during one round.

Every validator can find their validator public key, which is
different than the public keys / addresses that are assigned tokens,
via:

.. code-block:: console

  cat ~/.mycoind/config/priv_validator.json | jq .pub_key

If you still have the default genesis file from `tendermint init`,
this public key should match the one validator registered for this
blockchain, so it can mint blocks all by itself.

.. code-block:: console

  cat ~/.mycoind/config/genesis.json | jq .validators

In a multi-node network, all validators would have to generate their
validator key separately, then share the public keys, and forge
a genesis file will all the public keys present. Over two-thirds of
these nodes must be online, connected to the p2p network, and
acting correctly to mint new blocks. Up to one-third faulty nodes
can be tolerated without any problems, and larger numbers of nodes
usually halt the network, rather than fork it of mint incorrect
blocks.

The Tendermint dev team has produced
`a simple utility <https://github.com/tendermint/alpha>`__ to help
gather these keys.

Note that this liveness requirement means that after initializing
the genesis and starting up tendermint on every node, they must
set proper ``--p2p.seeds`` in order to connect all the nodes and
get enough signatures gathered to mint the first block.

HSMs
----

If we really care about security, clearly a plaintext file on our
machine is not the best solution, regardless of the firewall
we put on it. For this reason, the tendermint team is working
on integrating Hardware Security Modules (HSM) that will maintain
the private key secret on specialized hardware, much like
people use the Ledger Nano hardware wallet for cryptocurrencies.

This is under active development, but please check the following
repos to see the current state:

- `Signatory <https://github.com/tendermint/signatory>`__
  provides a rust api exposing many curves to sign with
- `YubiHSM <https://github.com/tendermint/yubihsm-rs>`__
  provides bindings to a YubiKey HSM
- `KMS <https://github.com/tendermint/kms>`__
  is a work in progress to connect these crates via sockets
  to a tendermint node.

It is too early now to try this technology, but it should be ready
for cosmos mainnet launch, and anyone building on weave should
follow these developments if they wish to have a secure validator
setup for their own mainnet launch.

Dynamic Validators
-------------------

A static validator set in the genesis file is quite useless for
a real network that is not just a testnet. Tendermint allows
the ABCI application to send back messages to update the validator
set at the end of every block. Weave-based applications can take
advantage of this and implement any algorithm they want to
select the validators, such as:

- `PoA <https://github.com/iov-one/weave/issues/32>`__
  where a set of keys (held by clients) can appoint the validators.
  This allows them to bring up and down machines, but the authority
  of the chain rests in a fixed group of individuals.
- ``PoS`` or proof-of-stake, where any individual can bond some of
  their tokens to an escrow for the right to select a validator.
  Each  validator has a voting power proportional to how much is
  staked. These staked tokens also receive some share of the block
  rewards as compensantion for the work and risk.
- ``DPoS`` where users can either bond tokens to their own
  validator, or "delegate" their tokens to a validator run by
  someone else. Everyone gets some share of the block rewards, but
  the people running the validator nodes typically take a
  commission on the delegated rewards, as they must perform real work.

For each of these general approaches there is a wide range
of tuning of incentives and punishments in order to acheive
the desired level of usability and security.

As extensions implementing any of these algorithms are implemented
as a weave extension, they should be linked to from here. Previous
related work from cosmos-sdk can be found in their
`simple stake <https://github.com/cosmos/cosmos-sdk/tree/master/x/simplestake>`__
implementation and the
`more complicated DPoS implementation <https://github.com/cosmos/cosmos-sdk/tree/master/x/stake>`__
with incentive mechanisms.
