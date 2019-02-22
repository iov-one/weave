/**
 Package errors implements custom error interfaces for weave.

  The idea is to reuse as many errors from this package as possible and define custom package
  errors when absolutely necessary. It is best to define a new error here if you feel it's going to
  be somewhat package-agnostic.

  x/multisig is a good package to take a look at in terms of usage with predefined strings with/without formatting.
  x/validators and x/sigs define some custom errors.

  If you want to register a custom error - use Register(code, description).
  For reusing errors - use Errxxx.New and Errxxx.Newf.

  Also, error package defines a convenient Is helper to compare errors, also each Error defines an Is
  helper to compare errors directly to that type.
 */
package errors
