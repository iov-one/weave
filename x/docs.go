/*
Package x contains some standard extensions

Extensions implement common functionality (Handler, Decorator,
etc.) and can be combined together to construct an application

All sub-packages are various extensions, useful to build
applications, but not necessary to use the framework.
All of them provide functionality commonly needed by blockchains.
You are welcome to import them if desired, but if they
don't match your particular needs, you may also write your
own extensions and use them instead.

Note that protobuf types in exported code will be prefixed by
the package, so follow standard go naming conventions and avoid
stutter. Use eg. `escrow.CreateMsg` in place of `escrow.CreateMsg`.
*/
package x
