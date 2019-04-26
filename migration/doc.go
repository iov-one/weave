package migration

/*

Package migration provides tooling necessary for working with schema versioned
entities. Functionality provided here can be applied both to messages and
models.


Global preparation
- update genesis to provide "migration" configuration
- register migration handlers using `RegisterRouters` function

Extension integration
- register your migrations functions in package `init`. Schema version is
declared per package not per entity so each upgrade must provide migration
function for all entities. Use `migration.NoModification` for those entities
that require no change,
- change your bucket to embed `migration.Bucket` instead of `orm.Bucket`
- wrap your handler with `migration.SchemaMigratingHandler`
- make sure `.Metadata.Schema` attribute of newly created messages is set. This
  is not necessary for models as it will default to the latest schema.

*/
