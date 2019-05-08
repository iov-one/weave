/*
Package errors implements custom error interfaces for weave.

Error declarations should be generic and cover broad range of cases. Each
returned error instance can wrap a generic error declaration to provide more
details.
Unless an error is very specific for an extension (ie ErrInvalidSequence in
x/sigs) it can be registered outside of the errors package.  To create a new
error istance use Register function. You must provide a unique, non zero error
code and a short description, for example:

  var ErrZeroDivision = errors.Register(9241, "zero division")

When returning an error, you can attach to it an additional context
information by using Wrap function, for example:

   func safeDiv(val, div int) (int, err) {
	   if div == 0 {
		   return 0, errors.Wrapf(ErrZeroDivision, "cannot divide %d", val)
	   }
	   return val / div, nil
   }

The first time an error instance is wrapped a stacktrace is attached as well.
Stacktrace information can be printed using %+v and %v formats.

*/

package errors
