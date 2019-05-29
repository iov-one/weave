----------------------
Configuring Tendermint
----------------------

Tendermint docs provide a `brief introduction <https://tendermint.com/docs/introduction/>`__
to the tendermint cli. By default all files are writen to
the ``~/.tendermint`` directory, unless you override that with
a different "HOME" directory by providing ``TMHOME=xyz`` or ``tendermint --home=xyz``.

When you call ``tendermint init``, it generates a ``config`` and ``data`` directory under the "HOME" dir. ``data`` will contain all blockchain
state as well as the application state. ``config`` will contain
configuration files. There are three main files to look at:

- ``genesis.json`` must be shared by all validators on a chain and is used to
  initialize the first block. We discuss this more in
  `Application Config <#application_config>`__
- ``config.toml`` is used to configure your local server, and can be
  configured much in the way the config for apache or postgres,
  to tune to your local system.
- ``priv_validator.json`` is used by any validating node to sign the blocks,
  and must be kept secret. We discuss this more in the
  `next section <./validators.html>`__.

Overriding Options
------------------

In general, any option you see in `the configuration file <https://tendermint.readthedocs.io/en/master/specification/configuration.html>`__
can also be provided via command-line or environmental variable.
It is a simple conversion:

Config:

.. code-block:: console

  [rpc]
  laddr = "tcp://0.0.0.0:8080"

Environment: ``export TM_RPC_LADDR=tcp://0.0.0.0:8080`` or ``export TMRPC_LADDR=tcp://0.0.0.0:8080`` (optional _ after TM)

Command line: ``tendermint --rpc.laddr=tcp://0.0.0.0:8080 ...``

Important Options
-----------------

There are many options to tune tendermint, but a few are quite
useful when configuring and deploying dev environements or testnets.
I will cover them here, but please take a longer look at
`all available options <https://github.com/tendermint/tendermint/blob/master/config/config.go>`__. I use the command line format
for these options, as it seems the most readable, but most of
these should be writen to the ``config.toml`` file or stored in
environmental options in the service ini (if using 12-factor style).

Dev:
- ``--p2p.upnp --proxy_app noop``: Don’t try to determine external address
  (`noop` for local testing)
- ``--log_level=p2p:info,evidence:debug,consensus:info,*:error``:
  Set the log levels for different subsystems (debug, info, error)
- ``--tx_index.index_all_tags=true`` to enable indexing for search
  and subscriptions. Should be on for public services,
  off for validators to conserve resources.
- ``--prof_laddr=tcp://127.0.0.1:7777`` to open up a profiling server
  at this port for debugging

Testnet:
- ``--moniker=billy-bob`` chooses a name to display on the node list
  to help understand the p2p network connections
- ``--mempool.recheck=false`` and ``--mempool.recheck_empty=false``
  limit rechecking all leftover tx in mempool, which can help
  throughput at the expense of possibly invalid tx making it into blocks
- ``--rpc.laddr=tcp://0.0.0.0:46657`` to change the interface or port
  we expose the rpc server (what we expose to the world)
- ``--p2p.laddr=tcp://0.0.0.0:46656`` to change the interface or port
  we expose the p2p server (what we use to connect to other nodes)
- ``--p2p.seeds=tcp://12.34.56.78:46656,tcp://33.44.55.66:46656``
  to set the seed nodes we connect to on startup to discover the
  rest of the p2p network
- ``p2p.pex=true`` turns on peer exchange, to allow us to
  dynamically update the network
- ``--consensus.create_empty_blocks=false`` to only create a block when
  there are tx (otherwise blockchain grows fast even with no activity)
- ``--consensus.create_empty_blocks_interval=300`` to create a block
  every 300s even if no tx
- ``--consensus.timeout_commit=5000`` to set block interval to 5s (5000ms)
  + time it takes to achieve consensus (which is generally quite small
  with < 20 or so well-connected validators)

Production:
- ``p2p.persistent_peers=tcp://77.77.77.77:46656`` contains peers we
  always remain connected to, regardless of peer exchange
- ``p2p.private_peer_ids=...`` contains peers we do not gossip.
  this is essential if we have a non-validating node acting as a
  buffer for a validating node
- ``--priv_validator_laddr=???`` to use a socket to connect to an
  hsm instead of using the priv_validator.json file

There are quite a few more options, but this is a good place to
get started, and you can dig in deeper once you see how these
numbers affect blockchains in practice.
