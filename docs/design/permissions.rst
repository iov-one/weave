---------------------------
Addresses and Authorization
---------------------------

When controlling the execution of a transaction, there are
two things to consider, authentication and authorization.
The first, authentication, deals with verifying who
is requesting the executions. The second, authorization,
deals with the access controls on the action, which can
refer to the authentication information.

Authentication
==============

Authentication information is added to the context as part
of the middleware stack, and used to verify the caller.
The simplest example is signature verification. We check
if the signature validates against a known public key, and
after checking nonces for replay protection, can authenticate
this public key for this transaction.

However, Ethereum devs are used to the concept of permissions
not just being tied to a signature, but potentially a smart
contract. We will allow something similar, but we don't need
to be as general, as we also don't have the same general
"anyone can call anything" architecture, nor do we run
untrusted code.

We use multiple middlewares to check for various conditions on
the transaction and add the authentication information to the
``Context``. The basic example, ``x/sigs.Middleware``, checks
if the Tx has signatures, and if so validates them.

.. literalinclude:: ../../x/sigs/tx.go
    :language: go
    :lines: 9-21

It stores the matching permissions in the context under a secret
key, and exposes an ``Authenticator`` that can be used to read
this information.

.. literalinclude:: ../../x/sigs/context.go
    :language: go
    :lines: 13-39

And finally, when we create a module that needs to read
authentication info, we can pass in the handler, so it can use
check authentication info from this middleware.

.. literalinclude:: ../../x/cash/handler.go
    :language: go
    :lines: 30-36

Note that this means that we don't let any extension authenticate
any action, but rather each extension can define which
other extensions it "trusts" to authenticate its actions.

Extending Authentication
------------------------

This system may look to complicated for just checking public key signatures, but it is designed to be flexible and allow
multiple authentication schemes. For example, if we want to
design HTLC, we could add an optional "Preimage" to the
Tx structure. We add a "Hasher" middleware that hashes
this preimage, and then grants the permission of something like
``preimage/<hash of preimage>``. This is stored in the context
and the "Hasher" exports an Authenticator that allows
access to this.

Once we build this "Hasher" extension, we can import it
into any other handler, to add the potential for hash preimages,
not just public key signatures, to trigger certain actions.
For some handlers this is not useful, so we can select it for
each handler. Of course, we don't want either/or, we often want
to support **both** authentication schemes. For this, there
is ``MulitAuth``, to combine them:

.. literalinclude:: ../../x/auth.go
    :language: go
    :lines: 16-26

Crypto-Conditions
-----------------

I am not the first one to try to build a generalized authentication
system for blockchain technology. Probably the most developed /
standardized proposal is Crypto-Conditions, which exists as
an `IETF Draft <https://tools.ietf.org/html/draft-thomas-crypto-conditions-02#section-7>`__
as well as `working implementations in multiple languages <https://github.com/rfcs/crypto-conditions>`__.

This area needs more research and we can either adopt them verbatim
or build a similar system. I have heard they are a bit difficult
to use, and also don't support some design choices we may want
(like using secp256k1 signatures, scrypt for hashing). But the
idea to have a general format to combine different conditions
in a boolean circuit is powerful. eg.
``(signature A and preimage H) OR Threshold(2, [Signature A, Signature B, Signature C])``.
We could use this to provide a very simple DSL for
defining multi-sig wallets, recovery phrases, etc.

Permissions
===========

*Authentication* defines who is requesting this transaction
and is added to the context as part of the middleware stack.
I will refer to the set of Authentications on a transaction as
the "requester", which may be made of signatures, preimages,
or other objects.

*Authorization* happens in a handler, and after it decides that
the are sufficient to execute this transaction. This is a
comparison of the "requester" with the required permissions

*Permissions* are a way to capture which conditions a transaction
must fulfill to be able to execute an action. They must be
serializable and can be stored along with an object.

The simplest example is "who can transfer money out of an account".
In many blockchains, they hash the public key and use that
to form an "address". Then this address is used as a primary key
to an account balance. A user can send tokens to any address,
and if I have signed with a public key, which hashes to the
"address" of this account, then I can authorize payments out of
the account. In Ethereum, smart contracts also have addresses and
can be the owner of tokens, not just signatures.

In this case, we can speak of a "transfer" permission on the
account. However, there doesn't have to be one permission for
an object. For example, an escrow may have a sender, a recipient,
and an arbiter. We could say sender and arbiter both have
permission to send the escrow to the recipient. The arbiter has
permission to return the escrow to the sender. And the recipient
has permission to update the recipient address. That is three
different permissions on one object, that can be checked based
on which action we are performing. And the "address" (primary
key) of the escrow object doesn't map to any of those three
permissions. (Otherwise, how can I create two escrows at once).

**TODO** Add scopes? More ideas?

Serialization
-------------

Now we have a clear understanding of what permissions are
in this context, and that there may be different permissions
on one object, we need to consider how to store them in the
database. A permission can be considered a tuple of
``(extension, type, data)``, for example a ed25519 public key
signature could be represented as ``("sigs", "ed25519", <addr>)``
and a sha256 hashlock could be ``("hash", "sha256", <hash>)``.

Note that the "data" doesn't need to reveal what the data that
will match this permission, but needs to be calculated from
it (eg. addr is first 20 bytes of a hash of the public key).
And each extension and type may have different interpretations
of the data.

If we enforce simple text for extension and type, we could
encode it as ``printf("%s/%s/%x", extension, type, data)``.
This is longer than the 20 bytes often used for addresses, and
maybe we could hash it first, but then we loose information.
I can envision a user wanting to know if an account is controlled
by a private key, a hash preimage, or another contract. If I am
going to set up an escrow with the same arbiter as you on another
chain to do atomic swap, I want to make sure that it is controlled
by a hash preimage (which you must reveal), not your private key
(which would not let me collect the other escrow).

Addresses
=========



We started with a simple address function, which was the first
20 bytes of the sha256 hash of a public key. However, this
left no room for smart contracts. Thus, we propose a simple
modification to this, promoting smart contracts to a first
class citizen: the first 20 bytes of the sha256 hash of
any key in the merkle store.

If the extension that is responsible for that key determines
that the tx matches the requirements stored in this key, then
this address is authorized for this transaction. Thus, the
public key check is a special case. The ``pubk`` extension
verifies signatures and checks sequence numbers to prevent
replays. The public key and current sequence are stored
under a key in the database, and that key is the source
of the address.

Since this is the most common address type, it should be
well-specified for external users. The pattern we use in
the standard modules is that all signatures must either
contain the public key, or the sha256 hash of the public
key (fingerprint). The sequence is stored in the merkle
tree under ``pubk:<fingerprint>``. When verifying the
signature and sequence number, we calculate the fingerprint
to load the proper sequence. When calculating the address
for a given public key (eg. to request payment), we
do the following:

::

    address := sha256("pubk:" || fingerprint)
    fingerprint := sha256(public_key_bytes)

Where ``||`` means concatenate, and ``public_key_bytes``
are the raw bytes of the public key.

Question: do we include the curve/algorithm the public key belongs
to in the fingerprint calculation? Is there any theoretical
collision here? How do we specify the type?

Questions: The sequence number (one/account, one/tx... define this well)
