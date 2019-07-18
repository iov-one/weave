# Changelog

## HEAD

## 0.19.0
- Remove `testify` dependency from our tests
- A new extension `x/cron` is added. It allows to configure weave application
  to be able to schedule messages for future execution.
- Proposals created with `x/gov` are having their tally executed automatically
  after the voting time is over. This is possible thanks to `x/cron` extension.
- Add `owner` index to bnsd `x/username` to be able to query tokens by owner.
- Allow empty targets in bnsd `x/username` to enable name reservation.
- Allow to update election quorum.
- Add self referencing `address` attribute to entities `aswap.Swap`,
  `escrow.Contract`, `distribution.Revenue`, `multisig.Contract` `gov.ElectionRule` and
  `paychan.PaymentChannel`.

Breaking changes

- `weave.Ticker` interface was updated.
- Add `owner` index to bnsd `x/username` to be able to query tokens by owner.
- Allow empty targets in bnsd `x/username` to enable name reservation.
- Username address type in `x/username` extension was changed from `[]byte` to
  `string`. Instead of base64 encoded value, a valid string is stored as the
  address.
- Some of the query paths in the `x/gov` package were updated to follow the
  naming convention.
- `gov.TallyMsg` is no longer available. Tally is created automatically when
  the voting time is over.



## 0.18.0

- `bnsd/x/username` genesis initializer implemented and included in `bnsd`.
- Support gov proposal vote, deletion and tally in `bnscli`
- Support gov proposal text resolution, update electorate, update election rules in `bnscli`
- Added `x/utils.ActionTagger`: all `bnsd` transactions now have
  `action=${msg.Path()}` tags. If there is a batch, there is one tag per
  sub-message. If it is a governance tally, the TallyMsg as well as the
  option (message executed on behalf of the governance stack) is tagged.

Breaking changes

- Unify all message paths to follow pattern `<package>/<message_name>`
- `app.Router` interface was changed. Handler registration requires a message
  and not message path.
- Unify all message attribute names
  - rename `src` and `sender` to `source`
  - rename `dst` and `recipient` to `destination`


## 0.17.0

- Unified dedupe logic for validator bookkeeping and validator diffs in `app`
- A new `errors.Field` was added. This allows to bind errors to field names and
  enables easier testing of group errors.
- Expose some more Genesis params to extension initializers. Utilise those in
  `x/validators` to store initial validator list and validate updates against
  this list while updating on every successful transaction.
- A new from scratch username implementation `x/username` was added. This
  implementation does not rely on `x/nft` package.
- Add `CommitInfo` to the context in order to be able to see who signed the
  current block.
- `cmd/bnscli` new commands
    - `with-fee` to configure a transaction fee,
    - `set-validators` to configure the validators,
    - `multisig` to create or update a multisig contract,
    - `with-multisig` to attach a multisig to a transaction,
    - `with-multisig-participant` to attach a participant to a multisig
      contract create/update transaction
- `x/aswap` allow timeout of a swap to be any value after 1970-01-01.
- `Iterator`s in store (btree cache and iavl adaptor) are now lazy. We also
  provide a `ReadOneFromIterator` function to easily get the first or last item
  in a range. This will only load desired items from disk and no longer greedily
  load the entire range before returning the first item.

Breaking changes

- Update `bnsd` transaction entities. All transaction attributes that point to
  a message are now snake case, and their naming follows the format
  `<package_name>_<message_type_name>`.
- Some messages were renamed to follow the general `start with a verb` format, also to remove stutter:
  - `cmd/bnsd`: `BatchMsg` -> `bnsd.ExecuteBatchMsg`, `ProposalBatchMsg` -> `bnsd.ExecuteProposalBatchMsg`
  - `x/aswap`: `CreateSwapMsg` -> `aswap.CreateMsg`, `ReleaseSwapMsg` -> `aswap.ReleaseMsg`, `ReturnSwapMsg` -> `aswap.ReturnMsg`
  - `x/cash`: `ConfigurationMsg` -> `cash.UpdateConfigurationMsg`
  - `x/currency`: `NewTokenInfoMsg` -> `currency.CreateMsg`
  - `x/distribution`: `NewRevenueMsg` -> `distribution.CreateMsg`, `ResetRevenueMsg` -> `distribution.ResetMsg`
  - `x/escrow`: `CreateEscrowMsg` -> `escrow.CreateMsg`, `ReleaseEscrowMsg` -> `escrow.ReleaseMsg`, `ReturnEscrowMsg` -> `escrow.ReturnMsg`, `UpdateEscrowPartiesMsg` -> `escrow.UpdatePartiesMsg`
  - `x/gov`: `TextResolutionMsg` -> `gov.CreateTextResolutionMsg`
  - `x/multisig`: `CreateContractMsg` -> `multisig.CreateMsg`, `UpdateContractMsg` -> `multisig.UpdateMsg`
  - `x/paychan`: `CreatePaymentChannelMsg` -> `paychan.CreateMsg`, `TransferPaymentChannelMsg` -> `paychan.TransferMsg`, `ClosePaymentChannelMsg` -> `paychan.CloseMsg`
  - `x/validators`: `SetValidatorsMsg` -> `validators.ApplyDiffMsg`
  - `bnsd/x/username`: `Username` string removed from all message names.
- `bnsd` specific protobuf objects (Tx, BatchMsg) are now under package `bnsd`, rather than
  conflicting with generic `app` messages in a namespace conflict.
- Moved some more messages from `x/validators` package to `weave`
- `cmd/bnsd`: `nft/username` allows now for any number of aliases/names for a
  single address. Lookup of the username by an address is no longer available.
- messages produced by `cmd/bnscli` have a new binary format incompatible with
  the previous version.
- `x/gov` added indexes to proposals and electorate to enable better client-side UX
- `cash.UpdateConfigurationMsg` requires `Metadata.Schema`
- ValidatorUpdate definitions now moved to `weave` package. Weave is using these definitions
now instead of abci internally.
- Simplified `Iterator` to 2 methods - Next() and Release()
- Removed `cmd/bcpd` application
- Removed `x/namecoin` package that is no longer used.
- Removed obsolete `examples` directory


## 0.16.0

- A new tool `cmd/bnscli` for interacting with a BNS node was created.
- Creation of a new proposal in `x/gov` extension is now restricted to only
  members of the electorate that this proposal is created for.
- Cleanup escrow: removed the support for atomic swap
- A new bucket implementation `orm.ModelBucket` was added that provides an
  easier to use interface when dealing with a single entity type.
- `migration` package was updated to provide `orm.ModelBucket` wrapper for
  transparent schema version migrations
- `x/gov` package was added, which maintains multiple versioned Electorates and
  ElectionRules that define voting rules (quorum, threshold, voting period) for
  a given Electorate. Votes can be tallied at the end and execute an
  application-defined action which is passed in to the constructor. This is
  compatible with standard handler interfaces and sample application-level
  setup is demonstrated in the test code (and all `sample*_test.go` code).
- Failed execution of the Proposal intent does not result in a failed
  transaction (so we update the Proposal state properly), but is rolled back
  independently and noted in `DeliverResult.Log` (reporting to be improved in a
  [future issue](https://github.com/iov-one/weave/issues/649))
- `x/gov` adds three internal transations: UpdateElectorate, UpdateElectionRule,
  and TextResolution. TextResolutions can only be created by elections and
  the text is stored in a bucket along with a reference to the electorate
  and proposal that they refer to.
- Enabled `x/batch` in bnsd. You can now send a batch of messages, which are
  executed atomically as one unit (all succeed, or no changes committed).
- `x/gov` methods are exposed in bnsd application. The list of messages that
  are eligible for proposals is in `cmd/bnsd/app/codec.go.ProposalOptions`.
  Note that you can also use a batch message with a subset of possible actions,
  to make multiple SendTx as part of a governance vote, for example.
- Dockerize all the protobuf tooling for easier developer experience and
  reproducible builds
- You can use "seq:multisig/usage/1" or similar in the genesis file to easily
  create addresses without manually encoding everything into 16 hex digits
- Introduce `errors.Append` function to combine errors into a new multi error
  type
- `spec` directory now contains protobuf files and testvectors (standard api
  objects in both json and binary encodings) to enable easier bindings and unit
  tests in client code, and projects that import weave.

Breaking changes

- Escrow does not support atomic swap anymore: preimage is removed from Tx and,
  haslock extension removed and arbiter now must be an Address and not a
  Condition
- `Metadata` attribute was removed from transaction attributes. This affects
  two entities `x/cash.FeeInfo` and `x/sigs.StdSignature`
- Max length of blockchain ids used in username NFTs is now 32 (previously 128)



## 0.15.0

- Tendermint is upgraded to version 0.31.5
- New `x/aswap` extension implementing atomic swap functionality. Atomic swap
  implementation is separated from `x/escrow`
- `x/cash` is using the new `gconf` package for configuration. New genesis path
  is used. To update genesis file, replace `"gconf": { "cash:xyz": "foo" }`
  with `"conf": { "cash": { "xyz": "foo" } }`
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
