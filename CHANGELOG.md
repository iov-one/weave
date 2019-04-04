# Changelog


## HEAD

- Tests were cleaned up and no use testify or convey packages. A new package
  `weavetest/assert` contains test helpers #249
- Simplify transaction message unpacking with `weave.LoadMsg` #437
- Initial version of governance model #450


## 0.13.0

- Allow app.ChainDecorators to accept nil #414
- Improve high-level benchmarks, sending coins with fees at ABCI level #408
- Remove composite literal uses of unkeyed fields #403
- Extend multisig contract with weights #285


## 0.12.1

- Cleanup coins package errors #378
- Add support for bech32 in genesis file #362

Breaking changes

- Distriubtion condition must match regexp for validation #425
- Deprecate Error.New for errors.Wrap #382
- Only support Error.Is with better algorithm #381
- Support time with escrow #392
