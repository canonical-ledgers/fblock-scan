package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

func Setup(conn *sqlite.Conn, speed bool) error {
	if err := checkOrSetApplicationID(conn); err != nil {
		return err
	}

	if err := enableForeignKeyChecks(conn); err != nil {
		return err
	}

	if err := applyMigrations(conn); err != nil {
		return err
	}

	if !speed {
		return nil
	}
	return optimizeSpeed(conn)
}

const ApplicationID int32 = 0x0FAC701D

func checkOrSetApplicationID(conn *sqlite.Conn) error {
	var appID int32
	if err := sqlitex.ExecTransient(conn, `PRAGMA "application_id";`,
		func(stmt *sqlite.Stmt) error {
			appID = stmt.ColumnInt32(0)
			return nil
		}); err != nil {
		return err
	}
	switch appID {
	case 0: // ApplicationID not set
		return sqlitex.ExecTransient(conn,
			fmt.Sprintf(`PRAGMA "application_id" = %v;`,
				ApplicationID),
			nil)
	case ApplicationID:
		return nil
	}
	return fmt.Errorf("invalid database: application_id")
}

func enableForeignKeyChecks(conn *sqlite.Conn) error {
	stmt, _, err := conn.PrepareTransient(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return err
	}
	defer stmt.Finalize()
	_, err = stmt.Step()
	return err
}

func optimizeSpeed(conn *sqlite.Conn) error {
	for _, sql := range []string{
		`PRAGMA synchronous = OFF;`,
		`PRAGMA journal_mode = MEMORY;`,
		`PRAGMA foreign_keys = OFF;`,
	} {
		stmt, _, err := conn.PrepareTransient(sql)
		if err != nil {
			return err
		}
		defer stmt.Finalize()
		if _, err := stmt.Step(); err != nil {
			return err
		}
	}
	return nil
}
