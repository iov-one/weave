# Changelog

## HEAD
- Cleanup escrow: removed the support for atomic swap
- A new bucket implementation `orm.ModelBucket` was added that provides an
  easier to use interface when dealing with a single entity type.
- `migration` package was updated to provide `orm.ModelBucket` wrapper for
  transparent schema version migrations
- `x/gov` package was added, which maintains multiple versioned Electorates
  and ElectionRules that define voting rules (quorum, threshold, voting period)
  for a given Electorate. Votes can be talliesd at the end and execute an
  application-defined action which is passed in to the constructor. This
  is compatible with standard handler interfaces and sample application-level
  setup is demonstrated in the test code (and all `sample*_test.go` code).
- Failed execution of the Proposal intent does not result in a failed transaction
  (so we update the Proposal state properly), but is rolled back independently
  and noted in `DeliverResult.Log` (reporting to be improved in
  a [future issue](https://github.com/iov-one/weave/issues/649))
- Enabled `x/batch` in bnsd. You can now send a batch of messages, which are
  executed atomically as one unit (all succeed, or no changes committed).
- `x/gov` methods are exposed in bnsd application. The list of messages that
  are eligible for proposals is in `cmd/bnsd/app/codec.go.ProposalOptions`.
  Note that you can also use a batch message with a subset of possible actions,
  to make multiple SendTx as part of a governance vote, for example.

Breaking changes  
- Escrow does not support atomic swap anymore: preimage is removed from Tx and, haslock extension removed
and arbiter now must be an Address and not a Condition 



## 0.15.0

- Tendermint is upgraded to version 0.31.5
- New `x/aswap` extension implementing atomic swap functionality. Atomic swap
  implementation is separated from `x/escrow`
- `x/cash` is using the new `gconf` package for configuration. New genesis path
  is used. To update genesis file, replace `"gconf": { "cash:xyz": "foo" }` with
  `"conf": { "cash": { "xyz": "foo" } }`
- Removed support for Go 1.10. Minimal required version is now 1.11.4.
- Added support for Go 1.12
- New `migration` package. Schema versioning for models and messages can be
  implemented by relying on functionality provided by this package.

Breaking changes

- Dependency management was migrated to Go modules. `dep` is no longer used or
  supported.
- `x/paychan` extension is using a wall clock for the timeout functionality
  instead of relying on the block height
- `gconf` package was reimplemented from scratch. Configuration can be changed
  during the runtime using messages.
- Many extensions where updated to provide `weave.Metadata` and support schema
  versioning as implemented by `migrations` package. Protobuf messages are
  using new schema and are not binary compatible with old ones. Updated
  extensions are: `x/cash`, `x/currency`, `x/distribution`, `x/escrow`,
  `x/msgfee`, `x/multisig`, `x/namecoin`,
  `x/nft`, `x/paychan`, `x/sigs`, `x/validators`


## 0.14.0

- Simplify transaction message unpacking with `weave.LoadMsg`
- Initial version of the governance extension (`x/gov`)
- Signature verification in `x/sigs` extension costs gas now
- A new message `BumpSequenceMsg` for incrementing a user sequence value in
  `x/sigs` extension
- A new validator subjective anti-spam fee was added

Breaking changes

- Timeout types in `x/escrow` changed to UNIX timestamps
- When considering expiration in `x/escrow` extension, expiration time is now
  inclusive

## 0.13.0

- Allow app.ChainDecorators to accept nil
- Improve high-level benchmarks, sending coins with fees at ABCI level
- Remove composite literal uses of unkeyed fields
- Extend multisig contract with weights


## 0.12.1

- Cleanup coins package errors
- Add support for bech32 in genesis file

Breaking changes

- Distribution condition must match regexp for validation
- Deprecate Error.New for errors.Wrap
- Only support Error.Is with better algorithm
- Support time with escrow
