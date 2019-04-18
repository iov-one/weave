# Changelog


## HEAD

- `gconf` package was reimplemented from scratch
- `x/cash` is using new `gconf` package for configuration. New genesis path is
  used.
- Tests were cleaned up and no use testify or convey packages. A new package
  `weavetest/assert` contains test helpers
- Simplify transaction message unpacking with `weave.LoadMsg`
- Initial version of governance model


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
