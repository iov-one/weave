--------
Querying
--------

Now that we have the blockchain running, let us connect
to it. Go back to the console where you were running
the `weave-js <https://github.com/confio/weave-js>`__
REPL shell and let's connect to this running chain.

``yarn cli demo-keys.db``

.. code:: javascript

    // default connects locally, give uri to connect remotely
    // let client = new Client("ws://54.246.248.147:46657")
    let client = new Client()
    await client.chainID()
    await client.status()
    await client.height()


Querying Accounts
-----------------

Once we have loaded a KeyPair into ``demo``, we can easily query
the balance with a helper that we use in the REPL:

.. code:: javascript

    let acct = await queryAccount(client, demo.address())
    pprint(acct)

Under the hood, this builds a query in the proper format to send
to the blockchain, then parses the response. The response is a
key-value pair, and we can extract the address from the key
(by removing the ``cash:`` prefix), and parse the value into
the actual balance as a protobuf object. 

We can look at what this helper function actually does:

.. code:: javascript

    // getAddr strings cash: prefix
    const getAddr = key => ({address: key.slice(5).toString('hex')});
    // query the /wallets path and use Set protobuf definition
    const queryAccount = (client, acct) =>
        client.queryParseOne(acct, "/wallets",
                             models.cash.Set, getAddr);
    // query the /wallets path and use UserData protobuf definition
    const querySigs = (client, acct) =>
        client.queryParseOne(acct, "/auth",
                             models.sigs.UserData, getAddr);


Interactive Help
----------------

The REPL also provides a way to query possible methods to call,
and easily inspect the objects to use. The goal is to allow a
developer to easily figure out how to use the various objects,
to get quick feedback before adding this code into his/her
application

.. code:: javascript

    .help
    .help queryAccount
    .help demo
    .help keys
    .help client

Explore a bit to see what you can do. When you feel comfortable
poking around, we will actually move the tokens.
