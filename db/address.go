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

func InsertAddresses(conn *sqlite.Conn, tx factom.Transaction,
	txID int64, whitelist map[factom.FAAddress]struct{}) (err error) {

	// If the tx does not contain an address in the whitelist, then we
	// rollback all changes.
	defer func(errP *error) {
		if *errP == ignoreErr {
			// Clear this err as it was only here to rollback the
			// changes.
			*errP = nil
		}
	}(&err)
	defer sqlitex.Save(conn)(&err)

	insertAdrTx := conn.Prep(`INSERT INTO "address_transaction"
                ("tx_id", "adr_id", "amount") VALUES
                (?, ?, ?)
                ON CONFLICT("tx_id", "adr_id") DO
                UPDATE SET "amount" = "amount" + "excluded"."amount";`)
	insertAdrTx.BindInt64(1, txID)

	var save bool
	if whitelist == nil { // Save all Addresses
		save = true
	}

	sign := int64(-1)
	for _, adrs := range [][]factom.AddressAmount{tx.FCTInputs, tx.FCTOutputs} {
		for _, adr := range adrs {
			amount := sign * int64(adr.Amount)
			adr := adr.FAAddress()
			if !save {
				_, save = whitelist[adr]
			}
			var adrID int64
			if adrID, err = AddressAdd(conn, &adr, amount); err != nil {
				return err
			}
			insertAdrTx.BindInt64(2, adrID)
			insertAdrTx.BindInt64(3, amount)
			if _, err = insertAdrTx.Step(); err != nil {
				return err
			}
			insertAdrTx.Reset()
		}
		sign = 1
	}

	if !save {
		// Rollback all changes.
		return ignoreErr
	}
	return nil
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
