/*
Package lateinit implements one time state initialization modification.

This functionality is helpful for configuring a running application, that
otherwise would be initialized by genesis.

Lateinit provides a way to execute exactly once a defined in code operation in
order to modify chain state by creating an entity. To apply such modification
two steps are requred.
1.1. Declare an initialization migration in code and build a new version of
     your application.
2.1. Distribute the new binary to all validators.
2.2. Send an ExecuteInitMsg with information which instructions to execute.

Once an inialization instruction is declared in code and executed, it must not
change and it must not be deleted. Immutability is necessary in order to build
the state by replaing the chain.

Each initialization can be executed only once and only for non existing keys.

*/
package lateinit
