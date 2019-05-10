---------------------------
Configuring the Application
---------------------------

The application is fed ``genesis.json`` the first time it starts up
via the ``InitChain`` ABCI message. There are three fields that
the application cares about:

- ``chain_id`` must be consistent on all nodes and distinct from all
  other blockchains. This is used in the tx signatures to provide replay
  protection from one chain and another
- ``validators`` are the initial set and should be stored if the app
  wishes to dynamically adjust the validator set
- ``app_state`` contains a map of data, to set up the initial blockchain
  state, such as initial balances and any accounts with special permissions.

App State
=========

If the backend ABCI app is weave-based, such as ``mycoind`` or ``bns``,
the app_state contains one key for each extension that it wishes
to initialize. Each element is an array of an extension-specific
format, which is fed into ``Initialized.FromGenesis`` from the
given extension.

Sample to set the balances of a few accounts:

.. code-block:: json

  "app_state": {
    "cash": [
      {
        "address": "849f0f5d8796f30fa95e8057f0ca596049112977",
        "coins": [ "88888888 BNS" ]
      },
      {
        "address": "9729455c431911c8da3f5745a251a6a399ccd6ed",
        "coins": [ "7777777.666666 IOV" ]
      }
    ]
  }

This format is application-specific and extremely important to set
the initial conditions of a blockchain, as the data is one of the
largest distinguishing factors of a chain and a fork.

``mycoind init`` will set up one account with a lot of tokens
of one name. For anything more complex, you will want to set this
up by hand. Note that you should make sure someone has saved
the private keys for all addresses or the tokens will never be
usable. Also, for cash, ticker must be 3 or 4 upper-case letters.
