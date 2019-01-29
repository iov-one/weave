/*

Package gconf implements a configuration store intended to be used as a global,
in-database configuration.

This package allows to load configuration from a genesis file and access it via
set of helper functions (`String`, `Int`, `Duration` etc).

Not being able to get a configuration value is a critical condition for the
application and there is no recovery path for the client. Application must be
terminated and configured correctly. This is why any failure results in a
panic.

*/
package gconf
