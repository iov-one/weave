--------------------
Posting Transactions
--------------------

So far we have focused on creating local keys and querying
the state of the blockchain. The **only** way to change the
state of the blockchain is to create a transaction and submit
it to the chain. If the ABCI app considers it valid, and it
is included in a block, it will be executed and update the
state.

That. is. the. **only**. way. to. change. the. blockchain.

No admin password, no direct inserts on the database,
no backdoors.... So we need to make these transactions bulletproof.
These are like HTTP POSTs but we use
`cryptographic signatures <../basics/authentication.html>`__
to authenticate them.

In the cli, there are wrappers to do this very easily. After
getting it to work, we will go through the steps in more
detail below, so you understand how this works, and how to
extend this to other transaction types you create for your
application.

We assume you have two keys `created earlier <./keys.html>`__
called demo and rcpt. We also assume that demo is the key
with tokens. Please check with ``keys.list()`` and
create them if needed, or change the names if you used
different names.


``yarn cli demo-keys.db``

.. code:: javascript

    // get the two accounts and the proper chain
    let demo = keys.get('demo')
    let rcpt = keys.get('rcpt')
    let client = new Client()
    let chainID = await client.chainID()
    // build the tx and submit it
    let tx = buildSendTx(demo, rcpt.address(), 12345, 'CASH', chainID);
    await client.sendTx(tx);
    await keys.save();

Query New Balance
-----------------

If ``sendTx`` was successful, you should be able to query
the accounts to see that money arrived. Check the balance,
send another tx, and see that it updated...

.. code:: javascript

    pprint(await queryAccount(client, demo.address()))
    pprint(await queryAccount(client, rcpt.address()))

Serialization
-------------

All the hard work was done in ``buildSendTx``. This is great
for using the chain and you can build your interfaces on top
of it. However, to enable you to extend the functionality and
add new transaction types, we will break down how it works.

The first step is to construct the unsigned transaction object.
This is a binary object and depends on the blockchain you
wish to use. As a default, all persistent objects in the weave
framework are define with protobuf and we prebuild a
protobuf definition of all models we use from the definitions
in the abci app (go code). We will cover that process in another
tutorial when you build your own app, but if you are curious,
you can try out ``yarn packProto``.

In the end we have model files defined with
`protobufjs <https://www.npmjs.com/package/protobufjs>`__
that are exposed to the REPL as ``models``, or can be directly
imported as ``let models = require('weave').weave``. Let's
take a look at these models and construct one...

.. code:: javascript

    // construct a coin
    let coin = {whole: 7890, fractional: 50000, ticker: 'CASH'}
    let pbCoin = models.x.Coin.create(coin);
    let bz = models.x.Coin.encode(pbCoin).finish();
    bz.length; // 13 bytes!
    // this makes a properly typed protobuf object
    let parsed = models.x.Coin.decode(bz);
    parsed;
    // or we can make a "normal" js object
    let obj = pbToObj(models.x.Coin, bz)
    obj;

Building Tx
-----------

We cover this in more detail when describing the weave framework,
but in short, we have a ``Msg`` (message) object that represents
the desired action (eg. ``SendMsg`` to send tokens). This message
is wrapped in a ``Tx``, which is an envelope to prove to the
blockchain that it should be accepted. Generally, this includes
the signature of the authorizing account, as well as any fees
to be paid to the miners.

In this example, we will just construct a simple unsigned
transaction.

.. code:: javascript

    let Coin = models.x.Coin
    let Tx = models.app.Tx
    let pay = Coin.create({whole: 3500, ticker: 'CASH'})
    let fee = Coin.create({whole: 1, ticker: 'CASH'})
    let msgObj = {src: demo.addressBytes(), dest: rcpt.addressBytes(), amount: pay}
    let msg = models.cash.SendMsg.create(msgObj)
    let finfo = models.cash.FeeInfo.create({fees: fee})
    let tx = Tx.create({sendMsg: msg, fees: finfo})
    pprint(tx)

Quite a few steps to set all the fields, which is why we build
helper functions to do this for us. But it should be clear
what is going on when we build it.

Signing Tx
----------

We already looked at signing bytes in the
`section on key management <./keys.html>`__.
When we sign a transaction, we:

* serialize the unsigned transaction
* sign those bytes
* create a footer with that information (including the public key of the signer and the sequence, so it can be verified by the blockchain)
* append that footer to the unsigned transaction object
* serialize that whole chunk

At the end is a sequence of bytes that can be parsed by the
blockchain and contains all information needed to verify the
signature and perform the desired action.

.. code:: javascript

    // create a signature
    let bz = Tx.encode(tx).finish();
    let {sig, seq} = demo.sign(bz, chainID);
    let std = models.sigs.StdSignature.create({pubKey: demo.pubkey, signature: sig, sequence: seq});
    pprint(std)
    // append it to the unsigned tx
    tx.signatures = [std]
    let signed = Tx.encode(tx).finish()
    signed.length
    // make sure to save the keys, as we updated the sequence
    seq
    demo.sequence
    await keys.save()


Searching for Tx
----------------

Let's submit this hand-built tx to the chain so you trust this
whole process above worked, and then we can query the entire
transaction history of our rich donor.

.. code:: javascript

    // find all tx that touched my account
    await client.sendTx(signed)
    let history = await client.searchParse("cash", demo.address(), Tx)
    history.length
    pprint(history[0])
    // find all tx that I signed....
    let signed = await client.searchParse("sigs", demo.address(), Tx)
    signed.length

**Homework**:
Now try querying the history of rcpt. It should be the for "cash",
but different for "sigs" (as demo was signing everything).
Maybe you can add a third of fourth account and try sending more
transactions, checking who sent what to what...

**Extra credit:**
Parse out those transactions to grab sender and recipient
and build up a local network of all transactions. Recursing
when one address interacts with  an address we have not seen
yet. You should be able to give any address that had one
transaction and follow it to build up the whole graph.
