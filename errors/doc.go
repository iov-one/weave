/*
Package errors implements custom error interfaces for weave.

The idea is to reuse as many errors from this package as possible and define custom package
errors when absolutely necessary. It is best to define a new error here if you feel it's going to
be somewhat package-agnostic.

x/multisig is a good package to take a look at in terms of usage with predefined strings with/without formatting.
x/validators and x/sigs define some custom errors.

If you want to register a custom error - use Register(code, description).
For reusing errors - use Errxxx.New and Errxxx.Newf.
Code stands for ABCI error code, which allows to distinguish types of errors
on the client side and act accordingly.

There is also support for stacktraces. Please ensure you create the custom error using
ErrXyz.New("...") or errors.Wrap(err, "...") at the point of creation to ensure we attach
a stacktrace. If you wrap multiple times, we only record the first wrap with the stacktrace.
(And don't do this as a global `var ErrFoo = errors.ErrInternal.New("foo")` or you will get a
useless stacktrace).

Once you have an error, you can use `fmt.Printf/Sprintf` to get more context for the error
	%s is just the error message
	%+v is the full stack trace
	%v appends a compressed [filename:line] where the error was created
	(source is wrappedError.Format)
Or call `err.StackTrace()` to get the raw call stack of the creation point
*/

package errors
