# Changelog

## HEAD


## 0.15.0

- Tendermint is upgraded to version 0.31.5
- New `x/aswap` extension implementing atomic swap functionality. Atomic swap
  implementation is separated from `x/escrow`
- `x/cash` is using the new `gconf` package for configuration. New genesis path
  is used. To update genesis file, replace `"gconf": { "cash:xyz": "foo" }` with
  `"conf": { "cash": { "xyz": "foo" } }`
- Removed support for Go 1.10. Minimal required version is not 1.11.4.
- Added support for Go 1.12

Breaking changes

- Dependency management was migrated to Go modules. `dep` is no longer used or
  supported.
- `x/paychan` extension is using a wall clock for the timeout functionality
  instead of relying on the block height
- `gconf` package was reimplemented from scratch. Configuration can be changed
  during the runtime using messages.


## 0.14.0

- Simplify transaction message unpacking with `weave.LoadMsg`
- Initial version of the governance extension (`x/gov`)
- Signature verification in `x/sigs` extension costs gas now
- A new message `BumpSequenceMsg` for incrementing a user sequence value in
  `x/sigs` extension
- New `migration` package. Schema versioning for models and messages can be
  implemented by relying on functionality provided by this package.

Breaking changes

- Many extensions where updated to provide `weave.Metadata` and support schema
  versioning as implemeneted by `migrations` package. Updated extensions are:
  x/cash`, `x/currency`, `x/distribution`, `x/escrow`, `x/msgfee`,
  `x/multisig`, `x/namecoin`, `x/nft`, `x/paychan`, `x/sigs`, `x/validators`


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
