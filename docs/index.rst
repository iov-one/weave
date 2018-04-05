.. Confio Weave documentation master file, created by
   sphinx-quickstart on Thu Apr  5 20:50:33 2018.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to Confio Weave's documentation!
========================================

`Confio Weave <https://github.com/confio/weave>`__
is a framework to quickly build your custom
`ABCI application <https://github.com/tendermint/abci>`__
to power a blockchain based on the best-of-class BFT Proof-of-stake
`Tendermint consensus engine <https://tendermint.com>`__.
It provides much commonly used functionality that can
quickly be imported in your custom chain, as well as a
simple framework for adding the custom functionality unique
to your project.

Blockchain Basics
-----------------

Some background material to help you get oriented with the
concepts behind blockchains in general and tendermint/weave
in particular. You can go through the mycoin tutorial without
reading this, but it is quite helpful to have a basic
understanding of these concepts before trying to build on weave.

.. toctree::
   :maxdepth: 2

   basics/blockchain.rst
   basics/consensus.rst
   basics/authentication.rst
   basics/state.rst

Mycoin Tutorial
-----------------

Weave comes with a simple cryptocurrency application,
``mycoin`` showing how to set up and use a blockchain with a
multi-currency wallet. This is the basis on which many
other applications can build and the simplest useful
example to understand the tooling

**TODO**

Weave Architecture
------------------

Once you understand the concepts and can run and interact
with a sample app, now it is time for you to extend the
codebase and write your own blockchain-based application.
Here is a primer to help you understand the architecture
and the various components you will use

.. toctree::
   :maxdepth: 2

   design/overview.rst
   design/queries.rst
   design/extensions.rst
   design/addresses.rst


Coding Tutorial
---------------

A step by step example of writing your first
application built on top of mycoind.

**TODO**

