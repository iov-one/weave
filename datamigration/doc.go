/*

Data migration package implements a functionality to declare and execute data
migrations.

Each data migration is a function registered under a unique name, that can be
executed only once.


To start using `datamigration` in your application, create a new file in your
application directory called `datamigrations.go`. Register all migrations in
the `init` function to ensure it is executed before the application starts
processing requests. For example:

	// cmd/myapp/datamigrations.go
	package main

	func init() {
		datamigration.MustRegister("register initial users", ...)
		// ...
		datamigration.MustRegister("create initial configuration entity", ...)
	}

In order to register a new data migration:
1. register a new migration in `init` using `datamigration.MustRegister`,
2. build a new binary of your application and make sure all validators are running it,
3. send a transaction with `ExecuteMigrationMsg`, signed by required addresses,
4. if migration was executed, it became a part of the chain. Code of the
   migration must remain unchanged forever.

Each migration is registered for a specific chain.

Migration must be a pure function. Migration function must depend only on the
data present in the context and database and not rely on non deterministic
input (i.e. `time.Now` or `rand` package).

A registered migration must not be deleted or altered. This is mandatory in
order for the state replying functionality.

*/
package datamigration
