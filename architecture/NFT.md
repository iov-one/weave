# NFT (Non-Fungible Token) Architecture Document

Weave has the concept of a fungible token. These are stored as
a token ticker and an amount, and any two values with the
same token ticker are considered fungible, that is they
can be combined and interchanged at will. That means we can
simply add 10.5 IOV with 4.75 IOV to get a balance of 15.25 IOV.

However, there are use-cases when every "token" is unique.
They have a unique id and a custom data payload. While you can
transfer ownership of the NFT, you cannot combine them.
One example is a domain name. I have a domain name `foo.com`
and you have a domain name `bar.com`. If I sell you my domain
name, you don't simply have 2 domains, you have `foo.com` and
`bar.com`. Each one of these domains not only has a unique id
(domain name), but also custom information, like DNS records
to point to a web site, mail server, etc. That can only be
modified by the owner.

We wish to use NFTs as the basis for human-readable addresses,
and blockchain lookup as part of BNS (the blockchain name service).
For that we need to design NFT support in weave, and in particular,
focusing on satisfying the use cases for BNS. Other expansions can
come later, but this gives is a useful target to create a MVP.

## Abstract ideas

TODO: link to Isabella's two docs as background

There are many types of NFTs, and each type will have it's own
rules, data format, and actions. This could be a domain name
or a crypto kitty. We will call this a `species` for now until
a better name comes up.

Any two tokens of the same species will behave by the same rules
and the species may have top-level rules of its own for issuing.
This is sort of like an extension in weave, or a smart contract
in ethereum. An given token is represented by some data stored
in the data-space of it's representative species. This format
is unspecified in the abstract, but each species will require
a specific data format. In weave term's, each `species` stores
the tokens in it's own specialized `Bucket`.

There is also some common data structure shared among tokens of
different species, which can be seen as an embedded struct that
many species extend. We can also write functions that work on
this generic fields, which can be reused among tokens of every species.

### Actions and Approvals

An NFT may have a number of custom actions. There are only
two-three generic ones: `issue` which is a constructor and
actually an action on the species, not a given token.
`transfer` which changes the owner of the token (this could
be a two-step sequence). and possibly `revoke` to destroy
a token, if there is some privileged judicial body.

But each species may have it's own special actions.
A human-name token with a fixed name may have a special
action `setPubkey` to update the public key associated
with the name NFT. A BNS token may have a few like
`setTrustedNodes`, `setRecentHeader`, `setTxCodec`, etc.
These will all update the species-specific payload.

The owner may not want to worry about updating the blockchain
nodes all the time. Especially when the owner is a 8 of 10
multisig contract for security. Or a DAO governance system.
Thus, we also want a way for them to delegate approval to
other actors for specific tasks. This should be fine-grained
to prevent abuse.

Of course, adding this concept of `approvals` means there is
now another generic action common across all species...
`grantApproval`.

## Interfaces

`id: []byte`: Every species will have it's own bucket and it's
own primary key, which will be unique. They will generally want
to add some validation logic as to what value ids are (such
as ascii alphanumeric).

`owner: Address`: Every token has one owner. This owner has full
control of the token, but can delegate other rights to other
accounts. We will like a secondary index on owner, so we can quickly
display all NFTs controlled by your private key.

`approvals: map[string][]Approval`: Every token may have approvals
for different actions. Actions are named strings and for each one,
there may be 0 or more accounts approved to execute it. We clearly
cannot store a map in the kvstore due to determinism concerns, so
I would suggest serializing it into arrays, such as 
`[]{name: string, approvals: []Approval}`, and then sorting the arrays
for consistency (top level by name string, each approval list sorted
by the approved account).

`payload: []byte/interface{}`: This is the species-specific part.
I will not discuss more here, just mention a term for later reference.

### Approval

There are various ideas for fields as part of the approval info.
The only one that is clear is account, which specifies who is approved.
There are concepts of some metadata that would also accompany it.

`account: Address`: The account which is approved for this action

`timeout: int` ? : The approval may automatically expire at some block height (or timestamp?). This will not trigger a cron job, but usage after the timeout will just error (and trigger a cleanup maybe?)

`immutable: boolean` ? : If ownership of the NFT is transferred,
it may be normal for all Approvals to be revoked, so the new owner
can re-issue them as needed. If I buy your domain name, I certainly
don't expect your sysadmins to be able to set my DNS records.
However, there are cases where the approval will persist beyond
a transfer of ownership (like a lease of mineral rights to some land).
The immutable flag could be used to renote the approval does not
expire upon transfer of ownership.

`count: int` ? : Another form of expiration is assigning a count of
the number of times this Approval can be used. It decrements by one
each use. `count = 1` is single-use. `count = 0 (or -1?)` could represent
infinite use. `count = 3` means three usages, etc...

Other ideas could be like "limit" if there is some function that has a 
numeric argument, as the max that can be used....

## Methods

Here is a sketch of some interfaces that may be used to
represent this in weave. Please adjust and experiment here...

```golang
// Payload is really a generic, let's just use an interface
// as placeholder, unless you have a better idea.
// All we know is that it must be serializable
type Payload Persistent

type Species interface {
  Issue(id []byte, owner Address, initialPayload Payload) NFT
  Load(id []byte) NFT
  // Revoke(id []byte)
}

// Note: we need to pass authorization info somehow,
// eg. via context or passed in explicitly 
type NFT interface {
  // read
  ID() []byte
  Owner() Address
  Approvals(action string) []Approval
  Payload() Payload

  // permissions
  Approve(action string, account Address, options ApprovalOptions)
  // RevokeApproval??

  // usage: params depend on action type
  TakeAction(actor Address, action string, params interface{})
  Transfer(newOwner Address) // ???? or maybe this is just an action?
}

type Approval struct {
  Account Address
  ApprovalOptions
}

type ApprovalOptions struct {
  Timeout int
  Count int
  Immutable bool
}
```

## Two-Step Transfer

Fungible tokens are currently designed so I can send them to anyone
without their approval, and they will appear in their wallet.
I mean, who doesn't want money? There are some issues with that
if you have to pay capital gains on the tokens, but this is generally
considered acceptable.

However, with NFTs there may be more use-cases where you need to 
explicitly approve taking ownership of an object (IRL you need to sign
a paper when I give you my car). As there may be liabilities or
responsibilities associated with ownership of a given NFT. Thus,
we also need to design for that case (while likely allowing the simpler
case).

In one-step case, you could imagine a "Transfer" action that the
owner can execute to set a new owner and possibly trigger a reset
of the Approvals.

In the two-step case, we could imagine the owner granting an
`immutable` (and likely `exclusive`) Approval to the new owner
to `acceptOwnership`. The new owner can now decide whether they want
to `takeAction(acceptOwnership)` to take ownership of the NFT.
We will need to make this immutable and/or exclusive to make sure
that we can do this safely. Eg. you offer to sell it to me, but after
giving it to me, you decide to give it to your friend as well,
who acceptsOwnership before me. We just need some design that can avoid
such race conditions.

## Use case: Human Address

The id is my username, maybe ascii alphanumeric, which is like
my email address.

The payload is a public key (or multiple, one per algorithm?)

The actions are `transfer` and `setPubkey`

## Use case: BNS

TODO