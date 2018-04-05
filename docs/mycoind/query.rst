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

    // load our key and query our balance
    let demo = keys.get('demo')
    demo.address()
    let acct = await queryAccount(client, demo.address())
    pprint(acct)

    // if you want to explore, use .help
    .help
    .help queryAccount
    .help demo
    .help keys
    .help client

Explore a bit to see what you can do. When you feel comfortable
poking around, we will actually move the tokens.
