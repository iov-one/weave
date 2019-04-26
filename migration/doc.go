package migration

/*

Package migration provides tooling necessary for working with schema versioned
entities. Functionality provided here can be applied both to messages and
models.


Global preparation.

- update application genesis to provide "migration" configuration. You can find
  documented configuration declaration in the protobuf declaration file,
- register migration messge handlers using `RegisterRouters` function
- register migration bucket query using `RegisterQuery` function


Extension integration.


- update all protobuf message declarations that are to be schema versioned.
  First attribute must be metadata. For example:

    import "github.com/iov-one/weave/codec.proto";
    message MyMessage {
      weave.Metadata metadata = 1;
      ...
    }

  Make sure that whenever you create a new entity, metadata attribute is
  provided as `nil` metadata value is not valid.
- register your migrations functions in package `init`. Schema version is
  declared per package not per entity so each upgrade must provide migration
  function for all entities. Use `migration.NoModification` for those entities
  that require no change. For example:

    func init() {
        func init() {
            migration.MustRegister(1, &MyModel{}, migration.NoModification)
            migration.MustRegister(1, &MyMessage{}, migration.NoModification)
        }
    }

- change your bucket implementation to embed `migration.Bucket` instead of
  `orm.Bucket`
- wrap your handler with `migration.SchemaMigratingHandler` to ensure all
  messages are always migrated to the latest schema before being passed to the
  handler,
- make sure `.Metadata.Schema` attribute of newly created messages is set. This
  is not necessary for models as it will default to the current schema version.

*/
