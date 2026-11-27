package sql

import "github.com/bestony/google-cloud-emulator/products/scaffold"

func init() {
	scaffold.SuppressServices(
		"google.cloud.sql.v1.SqlInstancesService",
		"google.cloud.sql.v1.SqlOperationsService",
		"google.cloud.sql.v1.SqlUsersService",
		"google.cloud.sql.v1.SqlDatabasesService",
		"google.cloud.sql.v1.SqlBackupRunsService",
		"google.cloud.sql.v1.SqlFlagsService",
		"google.cloud.sql.v1.SqlSslCertsService",
	)
}
