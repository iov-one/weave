----------------------
Extension Design (WIP)
----------------------

**State: Proposal**

This is a basic set of information on how to design an extension.

External Interface (Transactions)
=================================

Define a set of transactions that the module supports.
These should be generic as well, so another module may
support the same external calls, even with a very
different implementation.

**TODO**

Internal Calls
==============

Extensions need to call between each other, to trigger actions
in other "smart contracts", or query other state. This can be
done by importing them directly and linking them directly,
which is simple but rigid. Instead, we recommend to encapsulate
all internal calls we wish to make in an interface, which should
be passed in the constructor of the handler. And exporting
an object with methods that expose all functions we wish to
provide to other extensions.

When composing the application (in the main loop), we can
wire the extensions together as needed, while allowing the
compiler to verify that all needed functionality is properly
provided. A middleground between static-compile time resolution,
and dynamic run-time resolution.

**TODO** (This is like Controller???)

Persistent Models
=================

We suggest to create a ``.proto`` file defining the ``Msg`` structure
for any messages that this extension supports, as well as all data
structures that are supposed to be persisted in the merkle store.

**TODO: talk about ORM**

Wiring it up
============

We have 4 ways to connect this logic to the framework:

Handler, Decorator (Context), Ticker, Init

**TODO**

Flexible Composition
====================

How to connect them together?

We discussed a `flexible auth framework <./extensions.rst>`.
Here are some thoughts on how to flexibly tie together.

Using interfaces and inheritance... Allowing extensions to demand
functionality for what they link to, but not restricting the
implementation. This is a pattern that can be layered upon
the description in "Internal Calls" to allows for even more
flexibility in system composition.

**TODO: with examples**


