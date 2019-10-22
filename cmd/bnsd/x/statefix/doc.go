/*
Package statefix implements one time state modifications.

Statefix provides a way to execute once defined in code operations in order to
change the current state of the chain. To apply such modification two steps are
requred.
1. Declare a state migration in code and build new bnsd binary. Distribute the
   new binary to all validators.
2. Send an ExecuteFixMsg with information which fix to execute.

Once a fix is declared in code and executed, it must never change.

Each fix can be executed only once.

*/
package statefix
