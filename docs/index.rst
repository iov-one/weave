Welcome to IOV Weave's documentation!
========================================

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
charge more than the basic anti-spam fees. We recently added support for
"migrations" that can switch on modules, or enable logic updates, via
on-chain feature switch transactions.

Existing Modules
~~~~~~~~~~~~~~~~

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

In Progress
~~~~~~~~~~~

Light client proofs, custom token issuance and support for IBC (Inter Blockchain Communication) are currently being designed.

Basic Blockchain Terminology
----------------------------

If you are new to blockchains (or Tendermint), this is a
crash course in just enough theory to follow the rest of the setup.

.. toctree::
   :maxdepth: 2

   basics/blockchain.rst
   basics/consensus.rst
   basics/authentication.rst
   basics/state.rst

Running an Existing Application
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**TODO**

A good way to get familiar with setting up and running an application is to follow the steps in the `mycoin <mycoind/installation.html>`__ sample application. You can run this on your local machine.

To run a version of the IOV testnet on a cloud server, the steps to set up an instance on Digital Ocean are explored in this blog `post <https://medium.com/iov-internet-of-values/a-guide-to-deploy-a-validator-on-hugnet-3335192e11d5>`__
 
* Show how to query hugnet (and send tx) with iov-core (and generate addresses)
* (Also golang client?)
* Show to to compile bnsd, auto-init it, launch it with tendermint
* Query local application

Configuration
~~~~~~~~~~~~~

**TODO**

* Deep dive into the genesis file
* Show the init formats for multiple extensions
* Build custom init file
* Use your custom configuration

Understanding Weave
~~~~~~~~~~~~~~~~~~~

**TODO**
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

Custom App from Existing Modules
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**TODO**

* Look at application construction in examples/mycoind
* Look at more complex examples in cmd/bnsd
* Make your own app (first tutorial) with payment channels and multisig
* Compile and test the application
* Use custom golang client to perform a few actions
* Add custom iov-core connector
* Use the chain with iov-core


Coding a Custom Module (Tutorial Series)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**TODO**
Take much from *Backend Development Tutorial*

* Design 
* Implementation step by step

Additional Reading
~~~~~~~~~~~~~~~~~~

We are in the process of doing a large overhaul on the docs.
Until we are finshed, please look at the 
`older version of the docs <index_old.html>`__ for more complete (if outdated)
information
