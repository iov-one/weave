package txfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("txfee", r)

	r.Handle(&UpdateConfigurationMsg{},
		gconf.NewUpdateConfigurationHandler("txfee", &Configuration{}, auth, migration.CurrentAdmin))
}
