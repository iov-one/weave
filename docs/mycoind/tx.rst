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

We assume you have a profile and a key `created earlier <./keys.html>`__
created from an mnemonic. We assume this is the key
with tokens and Bert`s address the destination

.. code:: typescript

  const profile = new UserProfile();
  profile.addEntry(Ed25519SimpleAddressKeyringEntry.fromMnemonic("rose approve seek explain useful tomato canal ecology catch sad sign bracket hungry leave bacon clutch glide bundle control obey mandate creek mask faith"));
  const id1 = await profile.createIdentity(0);

  const writer = new IovWriter(profile);
  await writer.addChain(bnsConnector(TESTNET_RPC_URL));
  const chainId = writer.chainIds()[0];

  const bert = fromHex("e28ae9a6eb94fc88b73eb7cbd6b87bf93eb9bef0") as Address;

  const sendTx: SendTx = {
    kind: TransactionKind.Send,
    chainId: chainId,
    signer: id1.pubkey,  // this account must have money
    recipient: bert,
    memo: "My first transaction",
    amount: {
      whole: 1,
      fractional: 110000000,
      tokenTicker: "IOV" as TokenTicker,
    },
  };
  const result = await writer.signAndCommit(sendTx, 0);
  console.log("Tx submitted", result);

Query New Balance
-----------------

If ``sendTx`` was successful, you should be able to query
the accounts to see that money arrived. Check the balance,
send another tx, and see that it updated...

.. code:: typescript

  const account = await reader.getAccount({address: bert});
  console.log(account);
  console.log(account.data[0]);

Serialization
-------------

All the hard work was done in ``signAndCommit``. This is great
for using the chain and you can build your interfaces on top
of it. However, to enable you to extend the functionality and
add new transaction types, we will break down how it works.

The first step is to construct the unsigned transaction object.
This is a binary object and depends on the blockchain you
wish to use. As a default, all persistent objects in the iov-core
framework are define with protobuf and we prebuild a
protobuf definition of all models we use from the definitions
in the abci app (go code).

