Welcome to IOV Weave's documentation!
========================================

.. image:: ../_static/img/weave-logo.jpg
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

**Some highlights**

* Merkle-tree backed data store
* Highly configurable extension system, even for core logic like fees and signature validation
* Powerful customizations via genesis file
* Simple ORM on top of key-value store, with (proveable) secondary indexes
* Flexible permissioning system to use contracts as first-class actors
* "No empty blocks" for quick syncing on quiet chains
* Optional "product fees" for transactions that need to charge more than anti-spam

**Existing Modules**

* *cash* - wallets with multiple fungible tokens, fee deduction
* *sigs* - validate ed25519 signatures
* *multisig* - first-class multisig contracts, can modify membership
* *escrow* - Arbiter can safely hold tokens, or use with timeout for eg. vesting period
* *paychan* - Unidirectional payment channels, combine micro-payments with one on-chain settlement
* *distribution* - Safely distribute income (eg. fees) among multiple participants with flexible settings
* *batch* - Combine multiple transactions into one atomic operation (allow single-chain swap)
* *nft* - Generic NFT module
* *nft/username* - Part of cmd/bnsd, maps usernames to multiple chain-addresses, with reverse lookup
* *valdiators* - Update validator sets PoA style, by multisig or via on-chain Elections
* *msgfee* - Subjective minimum fees as quick anti-spam filter (set by each validator)
* *utils* - Features like KeyTagger, to make all db keys subscribe-able

**Coming soon**

* Elections - manage the submission of proposals, and the voting functionality (setting quorums, voting, vote counting)
* Smooth schema migrations / feature switches to easily allow hard and soft forks without stopping the chain
* Fully functional atomic swap module (currently made by joining escrow with a hashlock decorator)
* Light client proofs
* Custom token issuance
* IBC

Basic Blockchain Terminology
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

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
`older version of the docs <index_old.rst>`__ for more complete (if outdated)
information
