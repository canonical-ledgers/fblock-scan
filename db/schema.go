package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

const (
	// For the sake of simplicity, all chain DBs use the exact same schema,
	// regardless of whether they actually make use of the NFTokens tables.
	dbSchema = CreateTableFBlock +
		CreateTableAddress +
		CreateTableTransaction +
		CreateTableAddressTransaction

	currentDBVersion = 1
)

var migrations = []func(*sqlite.Conn) error{}

func applyMigrations(conn *sqlite.Conn) (err error) {
	empty, err := isEmpty(conn)
	if err != nil {
		return
	}
	if empty {
		if err = sqlitex.ExecScript(conn, dbSchema); err != nil {
			return
		}
		return updateDBVersion(conn)
	}

	version, err := getDBVersion(conn)
	if err != nil {
		return
	}
	if int(version)-1 == len(migrations) {
		return nil
	}
	if int(version)-1 > len(migrations) {
		return fmt.Errorf("no migration exists for DB version: %v", version)
	}

	// Always VACUUM after a successful migration.
	defer func() {
		if err != nil {
			return
		}
		stmt, _, err := conn.PrepareTransient(`VACUUM;`)
		if err != nil {
			panic(err)
		}
		defer stmt.Finalize()
		if _, err := stmt.Step(); err != nil {
			panic(err)
		}
	}()

	defer sqlitex.Save(conn)(&err)

	for i, migration := range migrations[version:] {
		version := int(version) + i
		fmt.Printf("running migration: %v -> %v\n", version, version+1)
		if err = migration(conn); err != nil {
			return
		}
	}
	return updateDBVersion(conn)
}

func isEmpty(conn *sqlite.Conn) (bool, error) {
	var count int
	err := sqlitex.ExecTransient(conn, `SELECT count(*) from "sqlite_master";`,
		func(stmt *sqlite.Stmt) error {
			count = stmt.ColumnInt(0)
			return nil
		})
	return count == 0, err
}

func getDBVersion(conn *sqlite.Conn) (int64, error) {
	var version int64
	err := sqlitex.ExecTransient(conn, `PRAGMA user_version;`,
		func(stmt *sqlite.Stmt) error {
			version = stmt.ColumnInt64(0)
			return nil
		})
	return version, err
}

func updateDBVersion(conn *sqlite.Conn) error {
	return sqlitex.ExecScript(conn, fmt.Sprintf(`PRAGMA user_version = %v;`,
		currentDBVersion))
}
