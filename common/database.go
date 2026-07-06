package common

type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypePostgreSQL DatabaseType = "postgres"
	DatabaseTypeClickHouse DatabaseType = "clickhouse"
)

var mainDatabaseType = DatabaseTypeSQLite
var logDatabaseType = DatabaseTypeSQLite

// Deprecated compatibility flags kept for legacy tests and older call sites.
var (
	UsingSQLite     = true
	UsingMySQL      = false
	UsingPostgreSQL = false
)

func syncLegacyDatabaseFlags(databaseType DatabaseType) {
	UsingSQLite = databaseType == DatabaseTypeSQLite
	UsingMySQL = databaseType == DatabaseTypeMySQL
	UsingPostgreSQL = databaseType == DatabaseTypePostgreSQL
}

func legacyMainDatabaseTypeOverride() (DatabaseType, bool) {
	count := 0
	var databaseType DatabaseType
	if UsingSQLite {
		count++
		databaseType = DatabaseTypeSQLite
	}
	if UsingMySQL {
		count++
		databaseType = DatabaseTypeMySQL
	}
	if UsingPostgreSQL {
		count++
		databaseType = DatabaseTypePostgreSQL
	}
	return databaseType, count == 1
}

func MainDatabaseType() DatabaseType {
	if databaseType, ok := legacyMainDatabaseTypeOverride(); ok {
		return databaseType
	}
	return mainDatabaseType
}

func LogDatabaseType() DatabaseType {
	return logDatabaseType
}

func SetMainDatabaseType(databaseType DatabaseType) {
	mainDatabaseType = databaseType
	syncLegacyDatabaseFlags(databaseType)
}

func SetLogDatabaseType(databaseType DatabaseType) {
	logDatabaseType = databaseType
}

func SetDatabaseTypes(mainType DatabaseType, logType DatabaseType) {
	mainDatabaseType = mainType
	logDatabaseType = logType
	syncLegacyDatabaseFlags(mainType)
}

func UsingMainDatabase(databaseType DatabaseType) bool {
	return MainDatabaseType() == databaseType
}

func UsingLogDatabase(databaseType DatabaseType) bool {
	return logDatabaseType == databaseType
}

var SQLitePath = "one-api.db?_busy_timeout=30000"

func init() {
	syncLegacyDatabaseFlags(mainDatabaseType)
}
