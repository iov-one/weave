/*
Package preregistration implements a storage for account preregistration. This
extension has no functionality beside being a storage for the preregistration
list. Once preregistration period is over, we will use state migration to
rewrite all preregistered names to the final account implementation.

This extension does not support schema migrations.
*/
package preregistration
