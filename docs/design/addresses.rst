---------------------------------
Addresses and Authorization (WIP)
---------------------------------

**State: Proposal**

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
contract. We should allow something similar. For example, we
could have a **role** middleware, that keeps a varying group
of public keys that can perform some action, as well as a
threshold number to sign. A wallet may be owned by this **role**,
and the members may be modified (on-chain) without modifying
the owning address. We wish to support such behavior seemlessly,
allowing general permissioning.

Currently, most objects are "owned" (ie. modifiable) by an address,
which is generally assumed to be the hash of a public key.
However, if we extend our concept of addresses to encompass
on-chain code as well, we gain a large amount of expressive power
and flexibility in our auth framework.

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

Authorization
=============

Each handler, when created, should take an ``AuthFunc`` as an
argument, to determine to check authentication information
for the given transaction. While the authentication information
is added to the context as part of the middleware stack,
authorization happens in a handler, and after it decides that
only address X is authorized to execute this transaction,
can then refer to the ``AuthFunc`` to verify if address X has
been authenticated for this transaction.

Each middleware registers its own ``AuthFunc`` and they can
be chained together. Thus, in the constructor of a handler,
one can specify which extensions we trust to provide
authentication info. Since each extensions is also associated
with a set of addresses, use of the addresses requires that
extension to be a trusted authentication provider, and to
have verified the access.

Add scopes? Modules checking context? More ideas?
