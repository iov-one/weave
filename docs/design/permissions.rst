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

However, Ethereum devs are used to the concept of authority
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
    :lines: 7-19

It stores the matching conditions in the context under a secret
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
this preimage, and then grants the condition of something like
``hash/sha256/<hash of preimage>``. This is stored in the context
and the "Hasher" exports an Authenticator that allows
access to this.

Once we build this "Hasher" extension, we can import it
into any other handler, to add the potential for hash preimages,
not just public key signatures, to grant certain permissions.
For some handlers this is not useful, so we can select it for
each handler. Of course, we don't want either/or, we often want
to support **both** authentication schemes. For this, there
is ``MulitAuth``, to combine them:

.. literalinclude:: ../../x/auth.go
    :language: go
    :lines: 19-29

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

Conditions
==========

*Authentication* defines who is requesting this transaction
and is added to the context as part of the middleware stack.
I will refer to the set of Authentications on a transaction as
the "requester", which may be made of signatures, preimages,
or other objects.

*Authorization* happens in a handler, where it decides if
the transaction can execute this transaction. It determines
if the transaction fulfills the necessary *conditions* to
assume the required *permission* to execute the action.

*Permission* is the right to perform a specific type of action,
like send tokens from an account, or release an escrow. These
permissions can be assigned to an individual, but more general
to a "Condition"

*Conditions* define what checks a transaction must fulfill to be
able to access a given permission. They must be serializable and
can be stored along with an object.

The simplest example is "who can transfer money out of an account".
In many blockchains, they hash the public key and use that
to form an "address". Then this address is used as a primary key
to an account balance. A user can send tokens to any address,
and if I have signed with a public key, which hashes to the
"address" of this account, then I can authorize payments out of
the account. In this case, the signature is *authentication*,
we must have *transfer permission* on this account, and the
*condition* is the presence of a signature with a public key
that hashes to the account's address.

In Ethereum, smart contracts also have addresses and can be
used as a condition, not just signatures. So, we can imagine
a variety of different conditions that can be required, not
just signatures. A hash preimage, the majority of votes
in an election, or presence of a merkle proof could be evaluated
by various middlewares and used as *conditions* to assume
given *permissions*. And one object / account could have
multiple different permissions.

Serialization
-------------

Now we have a clear understanding of what conditions are
in this context, and that there may be different conditions
on one object, we need to consider how to store them in the
database. A condition can be considered a tuple of
``(extension, type, data)``, for example a ed25519 public key
signature could be represented as ``("sigs", "ed25519", <addr>)``
and a sha256 hashlock could be ``("hash", "sha256", <hash>)``.

Note that the "data" doesn't need to reveal what the data is that
will match this condition, but needs to be calculated from
it (eg. addr is first 20 bytes of a hash of the public key).
And each extension and type may have different interpretations
of the data.

If we enforce simple text for extension and type, we could
encode it as ``sprintf("%s/%s/%X", extension, type, data)``.
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
left no room for other authentication mechanisms, and it
wasn't even clear how we could differentiate between ed25519
and secp256k1 signatures. However, the usecase was clear:
a short identifier that was uniquely tied to an authentication
condition, but *did not reveal* that condition.

We can redefine address to be the hash of a "condition",
not just public key bytes, and then we keep this functionality
while generalizing what a condition is.

::

    condition := sprintf("%s/%s/%X", extension, type, data)
    address := sha256(condition)[:20]

The questions is when and how to use each one. Any field that can
declare an owner must decide if those bytes represent a condition
or an address. The Authenticator can store fulfilled Conditions
in the Context, and then allow clients to check for matches either
by condition or by address. But where to use which one?

Here are some rough guidelines:

1. If we really need to save 20 bytes, use an *Address*. (But few places need that micro-optimization)
2. If we need visibility of control, use *Condition* (multi-sig solutions, arbiters, etc)
3. If you want to obscure control (until first use), use *Address*
4. Everything else, at your discretion, but prefer *Address* when possible for consistency.

I guess it is up to the extension developer, but I would generally
use Conditions for anything stored in the value and Address for
fields that appear in the key, unless there is a reason otherwise.

**Cash**: Key is Address

**Sigs**: Key is PublicKey (data section of Condition). We
construct a condition from it, then can compute the address.

**Escrow**: Sender and Receiver are Addresses, arbiter is defined by a Condition
in order to allow easy verification if it is a public key signature, a hash preimage,
or a multisig contract controlling the escrow.
