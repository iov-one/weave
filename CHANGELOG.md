# Changelog


## HEAD


- Added `migration` package. Schema versioning can be implemented by relying on
  functionality provided by this package.
- `gconf` package was reimplemented from scratch
- `x/cash` is using new `gconf` package for configuration. New genesis path is
  used. To update genesis file, replace "gconf": { "cash:xyz": "foo" }
  with "conf": { "cash": { "xyz": "foo" } }
- Tests were cleaned up and no use testify or convey packages. A new package
  `weavetest/assert` contains test helpers
- Simplify transaction message unpacking with `weave.LoadMsg`
- Initial version of governance model
- Introducing go modules instead of dep
- Removed support for go 1.10
- Added support for go 1.12
- Bumped minimum required version of go to 1.11.4+ as otherwise some commands
  fail because of go mod constraints

Breaking changes

- x/cash is using schema versioned model and messages
- x/paychan is using schema versioned model and messages
- x/escrow is using schema versioned model and messages
- x/distribution is using schema versioned model and messages
- x/namecoin is using schema versioned model and messages
- x/sigs is using schema versioned model and messages
- x/multisig is using schema versioned model and messages
- x/msgfee is using schema versioned model and messages
- x/currency is using schema versioned model and messages
- x/nft is using schema versioned model and messages


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
