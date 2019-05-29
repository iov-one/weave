======================================
Welcome to IOV Weave's documentation!
======================================

.. image:: _static/img/weave-logo.jpg
    :width: 800
    :alt: Weave Logo

`IOV Weave <https://github.com/iov-one/weave>`__
is a framework to quickly build your custom
`ABCI application <https://github.com/tendermint/abci>`__
to power a blockchain based on the best-of-class BFT Proof-of-stake
`Tendermint consensus engine <https://tendermint.com>`__.
It provides much commonly used functionality that can
quickly be imported in your custom chain, as well as a
simple framework for adding the custom functionality unique
to your project.

Some of the highlights of Weave include a Merkle-tree backed data store, 
a highly configurable extension system that also applies to the core logic such
as fees and signature validation. Weave also brings powerful customizations 
initialised from the genesis file. In addition there is a simple ORM 
which sits on top of a key-value store that also has proveable secondary indexes.
There is a flexible permissioning system to use contracts as first-class actors, 
“No empty blocks” for quick synchronizing on quiet chains, 
and the ability to introduce “product fees” for transactions that need to
charge more than the basic anti-spam fees. We have also added support for
"migrations" that can switch on modules, or enable logic updates, via
on-chain feature switch transactions.

Existing Modules
================

=================   =======================================================================================================================================
   Module             Description
=================   =======================================================================================================================================
Cash_                Wallets that support fungible tokens and fee deduction functionality
Sigs_                Validate ed25519 signatures
Multisig_            Supports first-class multiple signature contracts, and allow modification of membership
AtomicSwap_          Supports HTLC for cross-chain atomic swaps, according to the `IOV Atomic Swap Spec`_
Escrow_              The arbiter can safely hold tokens, or use with timeouts to release on vesting schedule
Governance_          Hold on-chain elections for text proposals, or directly modify application parameters
PaymentChannels_     Unidirectional payment channels, combine micro-payments with one on-chain settlement
Distribution_        Allows the safe distribution of income among multiple participants using configurations. This can be used to distribute fee income.
Batch_               Used for combining multiple transactions into one atomic operation. A powerful example is in creating single-chain swaps.
Validators_          Used in a PoA context to update the validator set using either multisig or the on-chain elections module
NFT_                 A generic Non Fungible Token module
NFT/Username_        Example nft used by bnsd. Maps usernames to multiple chain addresses, including reverse lookups
MessageFee_          Validator-subjective minimum fee module, designed as an anti-spam measure.
Utils_               A range of utility functions such as KeyTagger which is designed to enable subscriptions to database.
=================   =======================================================================================================================================

.. _Cash: https://github.com/iov-one/weave/tree/master/x/cash
.. _Sigs: https://github.com/iov-one/weave/tree/master/x/sigs
.. _Multisig: https://github.com/iov-one/weave/tree/master/x/multisig
.. _AtomicSwap: https://github.com/iov-one/weave/tree/master/x/aswap
.. _Escrow: https://github.com/iov-one/weave/tree/master/x/escrow
.. _Governance: https://github.com/iov-one/weave/tree/master/x/gov
.. _PaymentChannels: https://github.com/iov-one/weave/tree/master/x/paychan
.. _Distribution: https://github.com/iov-one/weave/tree/master/x/distribution
.. _Batch: https://github.com/iov-one/weave/tree/master/x/batch
.. _NFT: https://github.com/iov-one/weave/tree/master/x/nft
.. _Username: https://github.com/iov-one/weave/tree/master/cmd/bnsd/x/nft/username
.. _MessageFee: https://github.com/iov-one/weave/tree/master/x/msgfee
.. _Utils: https://github.com/iov-one/weave/tree/master/x/utils
.. _IOV Atomic Swap Spec: https://github.com/iov-one/iov-core/blob/master/docs/atomic-swap-protocol-v1.md

**In Progress**

Light client proofs, custom token issuance and support for IBC (Inter Blockchain Communication) are currently being designed.

Basic Blockchain Terminology
============================

.. toctree::
   :hidden:
   :maxdepth: 1

   basics/blockchain.rst
   basics/consensus.rst
   basics/authentication.rst
   basics/state.rst

If you are new to blockchains (or Tendermint), this is a
crash course in just enough theory to follow the rest of the setup.
`Read all <basics/blockchain.html>`__

Immutable Event Log
-------------------

If you are coming from working on typical databases, you can think
of the  blockchain as an immutable
`transaction log <https://en.wikipedia.org/wiki/Transaction_log>`__ .
If you have worked with
`Event Sourcing <https://martinfowler.com/eaaDev/EventSourcing.html>`__
you can consider a block as a set of events that can always be
replayed to create a `materialized view <https://docs.microsoft.com/en-us/azure/architecture/patterns/materialized-view>`__ .
Maybe you have a more theoretical background and recognize that a blockchain
is a fault tolerant form of
`state machine replication <https://en.wikipedia.org/wiki/State_machine_replication#Ordering_Inputs>`__ .
`Read more <basics/blockchain.html#immutable-event-log>`__

General Purpose Computer
-------------------------

Ethereum pioneered the second generation of blockchain, where they
realized that we didn't have to limit ourselves to handling payments,
but actually have a general purpose state machine.
`Read more <basics/blockchain.html#general-purpose-computer>`__

Next Generation
---------------

Since that time, many groups are working on "next generation" solutions
that take the learnings of Ethereum and attempt to build a highly scalable
and secure blockchain that can run general purpose programs.
`Read more <basics/blockchain.html#next-generation>`__

Eventual finality
-------------------------

All Proof-of-Work systems use eventual finality, where the resource cost
of creating a block is extremely high. After many blocks are gossiped,
the longest chain of blocks has the most work invested in it,
and thus is the true chain.
`Read more <basics/consensus.html#eventual-finality>`__

Immediate finality
-------------------------

An alternative approach used to guarantee constency comes out of
academic research into Byzantine Fault Tolerance from the 80s and 90s,
which "culminated" in `PBFT <http://pmg.csail.mit.edu/papers/osdi99.pdf>`__ .
`Read more <basics/consensus.html#immediate-finality>`__

Authentication
--------------

One interesting attribute of blockchains is that there are no
trusted nodes, and all transactions are publically visible
and can be copied.
`Read more <basics/authentication.html>`__

Upgrading the state machine
-------------------------

Of course, during the lifetime of the blockchain, we will want
to update the software and expand functionality. However,
the new software must also be able to re-run all transactions
since genesis.
`Read more <basics/state.html#upgrading-the-state-machine>`__

UTXO vs Account Model
-------------------------

There are two main models used to store the current state. 
The main model for bitcoin and similar chains is called UTXO, or Unspent transaction output. 
The account model creates one account per public key address and stores the information there. 
`Read more <basics/state.html#utxo-vs-account-model>`__

Merkle Proofs
--------------

Merkle trees are like binary trees, but hash the children at
each level. This allows us to provide a
`proof as a chain of hashes <https://www.certificate-transparency.org/log-proofs-work>`__.
`Read more <basics/state.html#merkle-proofs>`__


Running an Existing Application
===============================

.. toctree::
   :hidden:
   :maxdepth: 1

   mycoind/setup.rst
   mycoind/installation.rst
   mycoind/iovcore.rst

A good way to get familiar with setting up and running an application is to
follow the steps in the `mycoin <mycoind/installation.html>`__ sample application. 
You can run this on your local machine. If you don't have a modern Go development environment
already set up, please `follow these instructions <mycoind/setup.html>`__.

To connect a node to the BNS testnet on a cloud server, 
the steps to set up an instance on Digital Ocean are explored 
in this `blog post <https://medium.com/iov-internet-of-values/a-guide-to-deploy-a-validator-on-hugnet-3335192e11d5>`__.

Once you can run the blockchain, you will probably want to connect with it.
You can view a sample wallet app for the BNS testnet at https://wallet.hugnet.iov.one
Those that are comfortable with Javascript, should check out our
`IOV Core Library <mycoind/iovcore.html>`__ which allows easy access to the blockchain
from a browser or node environment.


Configuring your Blockchain
===========================

.. toctree::
   :hidden:
   :maxdepth: 1

   configuration/tendermint.rst
   configuration/application.rst
   configuration/validators.rst

When you ran the ``mycoind`` tutorial, you ran the following lines
to configure the blockchain:

.. code-block:: console

  tendermint init --home ~/.mycoind
  mycoind init CASH bech32:tiov1qrw95py2x7fzjw25euuqlj6dq6t0jahe7rh8wp

This is nice for automatic initialization for dev mode, but for
deploying a real network, we need to look under the hood and
figure out how to configure it manually.

Tendermint Configuration
------------------------

Tendermint docs provide a brief introduction to the tendermint cli.
Here we highlight some of the more important options and 
explain the interplay between cli flags, environmental variables,
and config files, which all provide a way to customize
the behavior of the tendermint daemon. 
`Read More <configuration/tendermint.html>`__

Application State Configuration
-------------------------------

The application is fed ``genesis.json`` the first time it starts up
via the ``InitChain`` ABCI message. There are three fields that
the application cares about: ``chain_id``, ``app_state``,
and ``validators``. To learn more about these fields
`Read More <configuration/application.html>`__

Setting the Validators
----------------------

Since Tendermint uses a traditional BFT algorithm to reach
consensus on blocks, signatures from specified validator keys
replace hashes used to mine blocks in typical PoW chains.
This also means that the selection of validators is an extremely
important part of the blockchain security.
`Read More <configuration/validators.html>`__


Building your own Application
=============================

.. toctree::
   :hidden:
   :maxdepth: 1

   design/overview.rst

Before we get into the strucutre of the application, there are
a few design principles for weave (but also tendermint apps in general)
that we must keep in mind.

Determinism
-----------

The big key to blockchain development is determinism.
Two binaries with the same state must **ALWAYS** produce
the same result when passed a given transaction.
`Read More <design/overview.html#determinism>`__

Abstract Block Chain Interface (ABCI)
-------------------------------------

To understand this design, you should first understand
what an ABCI application is and how that level blockchain
abstraction works. ABCI is the interface between the
tendermint daemon and the state machine that processes
the transactions, something akin to wsgi as the interface
between apache/nginx and a django application.
`Read More <design/overview.html#abci>`__

Persistence
-----------

All data structures that go over the wire (passed on any
external interface, or saved to the key value store),
must be able to be serialized and deserialized. An
application may have any custom binary format it wants,
although all standard weave extensions use protobuf.
`Read More <design/overview.html#persistence>`__

.. TODO: step through mycoind app top to bottom

.. TODO: tutorial with sample app (out of repo)

.. Understanding Weave
    Take much from *Backend Development Tutorial*
    * Working with Protobuf
    * Permission System and addresses
    * Models and Buckets
    * Messages and transactions
    * Migrations
    * Handlers and Decorators
    * Queries
    * Genesis Initialization
    * Error package

.. Custom App from Existing Modules
    * Look at application construction in examples/mycoind
    * Look at more complex examples in cmd/bnsd
    * Make your own app (first tutorial) with payment channels and multisig
    * Compile and test the application
    * Use custom golang client to perform a few actions
    * Add custom iov-core connector
    * Use the chain with iov-core


.. Coding a Custom Module (Tutorial Series)
    Take much from *Backend Development Tutorial*
    * Design 
    * Implementation step by step

.. Deploying the code -> where does that go?



Additional Reading
==================

We are in the process of doing a large overhaul on the docs.
Until we are finished, please look at the
`older version of the docs <index_old.html>`__ for more complete (if outdated)
information
