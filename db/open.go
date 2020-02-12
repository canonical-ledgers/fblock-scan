package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

func Setup(conn *sqlite.Conn) error {
	if err := checkOrSetApplicationID(conn); err != nil {
		return err
	}

	if err := applyMigrations(conn); err != nil {
		return err
	}

	if err := enableForeignKeyChecks(conn); err != nil {
		return err
	}
	return nil
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
	return sqlitex.ExecScript(conn, `PRAGMA foreign_keys = ON;`)
}
