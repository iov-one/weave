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

* Elections
* Fully functional atomic swap module (currently made by joining escrow with a hashlock decorator)
* Light client proofs
* Custom token issuance
* IBC

**More information**

.. toctree::
   :maxdepth: 2

   index_old.rst
