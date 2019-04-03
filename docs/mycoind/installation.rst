------------
Installation
------------

To run our system, we need three components:

* ``mycoind``, our custom ABCI application
* ``tendermint``, a powerful blockchain consensus engine
* ``iov-core``, a generic typescript client

If you have never used tendermint before, you should
read the `ABCI Overview <https://tendermint.com/docs/introduction/introduction.html#abci-overview>`__
and ideally through to the bottom of the page. The end result
is that we have three programs communicating:

::

    +---------+            +------------+                    +----------+
    | mycoind  | <- ABCI -> | Tendermint   |  <- websocket ->  | iov-core  |
    +---------+            +------------+                    +----------+

``mycoind`` and ``tendermint`` run on the same computer and communicate via
a binary protocol over localhost or a unix socket. Together they form
a "blockchain". In a real setup, you would have dozens (or hundreds)
of computers running this backend communicating over a self-adjusting
p2p gossip network to replicate the state. For application development
(and demos) one copy will work, but has none of the fault tolerance of a
real blockchain.

``iov-core`` is a typescript library to communicate with the blockchain,
perform binary encoding/decoding of data types, and manage private
keys locally to sign transactions. This library is designed to be imported
by your application to power an eg. electron app.

Install backend programs
========================

You should have a proper go development environment, as explained
in the `last section <./installation.html>`__. Now, check out
the most recent version of iov-one/weave and build ``mycoind`` then get
the version 0.21 for ``tendermint`` from `here <https://github.com/tendermint/tendermint/releases?after=v0.22.0>`__.
You can also build ``tendermint`` from source following the instructions
`there <https://github.com/tendermint/tendermint/blob/master/docs/introduction/install.md>`__
but make sure to use the tag **0.21** as other versions might not be compatible.

.. code:: console

    go get github.com/iov-one/weave
    cd $GOPATH/src/github.com/iov-one/weave
    make deps
    make install
    # test it built properly
    tendermint version
    # 0.21.0-46369a1a
    mycoind version
    # v0.7.0

Those were the most recent versions as of the time of the writing,
your code should be a similar version. If you have an old version
of the code, you may have to delete it to force go to rebuild:

.. code:: console

    rm `which tendermint`


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

Note: if you did anything funky during setup and managed to get yourself a rogue tendermint
node running in the background, you might encounter errors like `panic: Error initializing DB: resource temporarily unavailable`.
A quick ``killall tendermint`` should get you back on track.
