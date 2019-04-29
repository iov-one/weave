/*

Package gconf provides a toolset for managing an extension configuration.

Extension that defines a configuration object can use gconf package to load
initial state from genesis, update configuration state via messages and to
retrieve configuration object from the store.

Each extension can declare and store only one configuration object.

To use gconf you must follow a few simple principles.

1. Define your configuration as a protobuf message.

2. Define your configuration update message as a protobuf message. It must have
a `patch` field that holds the new configuration state.

3. Zero field values are ignored during the update message processing,

4. use `InitConfig` inside of your extension initializer to copy configuration
from the genesis into the database,

5. Use `Load` function to load your configuration state from the database,


See existing extensions for an example of how to use this package.

*/
package gconf
