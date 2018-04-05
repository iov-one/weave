------------
Installation
------------

To run our system, we need three components:

* ``mycoind``, our custom ABCI application
* ``tendermint``, a powerful blockchain consensus engine
* ``weave-js``, a generic javascript client

If you have never used tendermint before, you should
read the `ABCI Overview <http://tendermint.readthedocs.io/en/master/introduction.html#abci-overview>`__
and ideally through to the bottom of the page. The end result
is that we have three programs communicating:

.. code-block::

    +---------+            +------------+                   +----------+
    | mycoind | <- ABCI -> | Tendermint |  <- websocket ->  | weave-js |
    +---------+            +------------+                   +----------+

``mycoind`` and ``tendermint`` run on the same computer and communicate via
a binary protocol over localhost or a unix socket. Together they form
a "blockchain". In a real setup, you would have dozens (or hundreds)
of computers running this backend communicating over a self-adjusting
p2p gossip network to replicate the state. For application development
(and demos) one copy will work, but has none of the fault tolerance of a
real blockchain.

``weave-js`` is a javascript library to communicate with the blockchain,
perform binary encoding/decoding of data types, and manage private
keys locally to sign transactions. This library is designed to be imported
by your application to power an eg. electron app. We also provide an
interactive REPL shell with helper functions pre-loaded, so developers
can interact with the blockchain and get a better feel for the conepts.
Until there is a great demo UI (hint, hint), we will use weave-js
to demonstrate the capabilities.

Install backend programs
========================

You should have a proper go development environment, as explained
in the `last section <./installation.html>`__. Now, check out
the most recent version of confio/weave and build both
``mycoind`` and ``tendermint``.

.. code:: console

    go get github.com/confio/weave
    cd $GOPATH/src/github.com/confio/weave
    make deps
    make install

    # test it built properly
    tendermint version
    # 0.17.1-6f995699
    mycoind version
    # v0.2.1-21-g35d9c08

Those were the most recent versions as of the time of the writing,
your code should be a similar version. If you have an old version
of the code, you may have to delete it to force go to rebuild:
``rm `which tendermint` ``.



Install client cli
==================

Node is much less picky as to where the code lives, so just
find a nice place to store the client code, and install
a copy:

.. code:: console

    git clone https://github.com/confio/weave-js.git
    cd weave-js
    yarn install  # you did set this up earlier, right?

After a bit, it should have pulled down all the required
node modules. Let's make sure it runs, and set up your first
key pair while we are at it.

``yarn cli demo-keys.db``

.. code:: javascript

    let demo = keys.add('demo')
    await keys.save()
    keys.list()

Exit the shell and start up the cli again, make sure the same key
is still there when you type ``keys.list()`` (auto-loaded from our local db).
How, save this hex address, you will need it for the next step,
initializing the blockchain. How else will you give yourself
access to endless tokens?

Initialize the Blockchain
=========================

Before we start the blockchain, we need to set up the initial state.
This is defined in a genesis block. Both ``tendermint`` and ``mycoind``
have a directory to store configuration and internal database state.
By default those are ``~/.tendermint`` and ``~/.mycoind``. However, to
make things simpler, we will ask them both to put everything in the
same directory.

First, we create a default genesis file, the private key for the
validator to sign blocks, and a default config file.

.. code:: console

    # make sure you really don't care what was in this directory and...
    rm -rf ~/.mycoind
    tendermint init --home ~/.mycoind

You can take a look in this directory if you are curious. The most
important piece for us is ``~/.mycoind/config/genesis.json``.
You may also notice ``~/.mycoind/config/config.toml`` with lots
of `options to set <https://tendermint.readthedocs.io/en/master/using-tendermint.html#configuration>`__ for power users.

We want to add a bunch of tokens to the account we just made before
launching the blockchain. And we'd also like to enable the indexer,
so we can search for our transactions by id (default state is off).
But rather than have you fiddle with the config files by hand,
you can just run this to do the setup:

.. code:: console

    mycoind init CASH <hex address from above>

Make sure you enter the same hex address, this account gets the tokens.
You can take another look at ``~/.mycoind/config/genesis.json`` after running
this command. The important change was to "app_state". You can also
create this by hand later to give many people starting balances, but let's
keep it simple for now and get something working. Feel free to
wipe out the directory later and reinitialize another blockchain with
custom configuration to experiment.

Start the Blockchain
====================

We have a private key and setup all the configuration.
The only thing left is to start this blockchain running.

.. code:: console

    tendermint node --home ~/.mycoind --p2p.skip_upnp > ~/.mycoind/tendermint.log &
    mycoind start

After a few seconds this should start seeing "Commit Synced" messages.
That means the blockchain is working away and producing new blocks,
one a second.
