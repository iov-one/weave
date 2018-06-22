-----------------
Defining Messages
-----------------

We just discussed messages, which are persistent objects
requiring validation, which are stored in our local
key-value store. Messages are requests for a change in the
state, the action part of a transaction. They also need
to be persisted (to be sent over the wire and stored on the
blockchain), and must also be validated. They later are passed
into `Handlers <https://godoc.org/github.com/confio/weave#Handler>`_
to be processed and effect change in the blockchain state.

Messages vs. Transactions
-------------------------

A message is a request to make change and this is the basic
element of a blockchain. A transaction contains a message
along with metadata and authorization information, such
as fees, signatures, nonces, and time-to-live.

A `Transaction <https://godoc.org/github.com/confio/weave#Tx>`_
is fundamentally defined as anything persistent that holds a message:

.. code:: go

    type Tx interface {
        Persistent
        // GetMsg returns the action we wish to communicate
        GetMsg() (Msg, error)
    }

And every application can extend it with additional functionality,
such as
`Signatures <https://godoc.org/github.com/confio/weave/x/sigs#SignedTx>`_,
`Fees <https://godoc.org/github.com/confio/weave/x/cash#FeeTx>`_,
or anything else your application needs. The data placed in the
Transaction is meant to be anything that applies to all modules, and
is processed by a Middleware.

A `Message <https://godoc.org/github.com/confio/weave#Msg>`_
is also persistent and can be pretty much anything that an
extension defines, as it also defines the
`Handler <https://godoc.org/github.com/confio/weave#Handler>`_
to process it. The only necessary feature of a Message is
that it can return a ``Path() string`` which allows us to
route it to the proper Handler.

When we define a concrete transaction type for one application,
we define it in protobuf with a set of possible messages that
it can contain. Every application can add optional field to the
transaction and allow a different set of messages, and the
Handlers and Decorators work orthogonally to this, regardless
of the concrete Transaction type.

Defining Messages
-----------------

Messages are similar to the ``POST`` endpoints in a typical
API. They are the only way to effect a change in the system.
Ignoring the issue of authentication and rate limitation,
which is handled by the Decorators / Middleware, when we design
Messages, we focus on all possible state transitions and the
information they need to proceed.

In the blog example, we can imagine:

* Create Blog
* Update Blog Title
* Add/Remove Blog Author
* Create Post
* Create Profile
* Modify Profile (which may be merged with above)

We can create a protobuf message for each of these types:

.. literal-include

And then add a ``Path`` method that returns a constant based on
the type:

.. literal-include

Validation
----------

While validation for data models is much more like SQL constraints:
"max length 20", "not null", "constaint foo > 3", validation for
messages is validating potentially malicious data coming in from
external sources and should be validated more thoroughly.
One may want to use regexp to avoid control characters or null bytes
in a "string" input. Maybe restrict it to alphanumeric or ascii
characters, strip out html, or allow full utf-8. Addresses must be
checked to be the valid length. Amount being sent to be positive
(else I send you -5 ETH and we have a TakeMsg, instead of SendMsg).

The validation on Messages should be a lot more thorough and well
tested than the validation on data models, which is as much documentation
of acceptable values as it is runtime security.

**TODO**

* Messages vs. Transactions, what is the distinction?
* Defining message types - needed data, non-maleability
* Validation of Messages
