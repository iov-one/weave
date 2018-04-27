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

Mycoin Tutorial
-----------------

Weave comes with a simple cryptocurrency application,
``mycoin`` showing how to set up and use a blockchain with a
multi-currency wallet. This is the basis on which many
other applications can build and the simplest useful
example to understand the tooling. For all those who like
learning by doing, this will help you understand the power
of the framework

.. toctree::
   :maxdepth: 2

   mycoind/setup.rst
   mycoind/installation.rst
   mycoind/keys.rst
   mycoind/query.rst
   mycoind/tx.rst
   mycoind/events.rst

Blockchain Basics
-----------------

Some background material to help you get oriented with the
concepts behind blockchains in general and tendermint/weave
in particular. It is quite helpful to have a basic
understanding of these concepts before trying to build on weave.

.. toctree::
   :maxdepth: 2

   basics/blockchain.rst
   basics/consensus.rst
   basics/authentication.rst
   basics/state.rst

Deployment
----------

A brief introduction into how to deploy a blockchain app.
Once you compile the code, hwo do you run it?

.. toctree::
   :maxdepth: 2

   deployment/configuration.rst
   deployment/validators.rst
   deployment/tooling.rst


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
   design/permissions.rst
   design/queries.rst
   design/extensions.rst


Backend Development
-------------------

A step by step example of writing your first
application built on top of mycoind.
This is all about writing go code that runs as an ABCI app.
We will write a new extension and compile an application
that builds upon ``mycoind``.

**TODO**

