package db

import (
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/Factom-Asset-Tokens/factom"
)

// CreateTableAddress is a SQL string that creates the "address" table.
const CreateTableAddress = `CREATE TABLE "address" (
        "address" TEXT PRIMARY KEY NOT NULL,
        "balance" INTEGER NOT NULL,
        "memo"    TEXT
);
`

// Add adds add to the balance of adr, creating a new row in "address" if it
// does not exist. If successful, the rowid of adr is returned.
func AddressAdd(conn *sqlite.Conn, adr *factom.FAAddress, add int64) (int64, error) {
	stmt := conn.Prep(`INSERT INTO "address"
                ("address", "balance") VALUES (?, ?)
                ON CONFLICT("address") DO
                UPDATE SET "balance" = "balance" + "excluded"."balance";`)
	stmt.BindText(1, adr.String())
	stmt.BindInt64(2, add)
	_, err := stmt.Step()
	if err != nil {
		return -1, err
	}
	return SelectAddressID(conn, adr)
}

const sqlitexNoResultsErr = "sqlite: statement has no results"

// SelectIDBalance returns the rowid and balance for the given adr.
func SelectAddressIDBalance(conn *sqlite.Conn,
	adr *factom.FAAddress) (adrID int64, bal uint64, err error) {
	adrID = -1
	stmt := conn.Prep(`SELECT "rowid", "balance" FROM "address" WHERE "address" = ?;`)
	defer stmt.Reset()
	stmt.BindText(1, adr.String())
	hasRow, err := stmt.Step()
	if err != nil {
		return
	}
	if !hasRow {
		return
	}
	adrID = stmt.ColumnInt64(0)
	bal = uint64(stmt.ColumnInt64(1))
	return
}

// SelectAddressID returns the rowid for the given adr.
func SelectAddressID(conn *sqlite.Conn, adr *factom.FAAddress) (int64, error) {
	stmt := conn.Prep(`SELECT "rowid" FROM "address" WHERE "address" = ?;`)
	stmt.BindText(1, adr.String())
	return sqlitex.ResultInt64(stmt)
}

// SelectAddressCount returns the number of rows in "address". If nonZeroOnly
// is true, then only count the address with a non zero balance.
func SelectAddressCount(conn *sqlite.Conn, nonZeroOnly bool) (int64, error) {
	stmt := conn.Prep(`SELECT count(*) FROM "address" WHERE "id" != 1
                AND (? OR "balance" > 0);`)
	stmt.BindBool(1, !nonZeroOnly)
	return sqlitex.ResultInt64(stmt)
}
