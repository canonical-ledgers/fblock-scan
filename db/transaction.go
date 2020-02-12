package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/Factom-Asset-Tokens/factom"
)

// CreateTableTransaction is the SQL that creates the "transaction" table which
// contains metadata about individual transactions within an FBlock.
const CreateTableTransaction = `CREATE TABLE "transaction" (
        "height" INT NOT NULL,    -- "fblock"."height"

        "fb_offset" INT NOT NULL, -- index of tx data within "fblock"."data"
        "size" INT NOT NULL, -- length of tx data in bytes

        "id" BLOB PRIMARY KEY NOT NULL, -- hash of tx ledger data
        "timestamp" INT NOT NULL,

        -- amounts
        "total_fct_in"  INT NOT NULL, -- denoted in factoshis
        "total_fct_out" INT NOT NULL, -- denoted in factoshis
        "total_ec_out"  INT NOT NULL, -- denoted in factoshis

        "memo" TEXT,

        FOREIGN KEY("height") REFERENCES "fblock"("height")
);
`

const CreateTableAddressTransaction = `CREATE TABLE "address_transaction" (
        "tx_id" INT NOT NULL,  -- "transaction"."rowid"
        "adr_id" INT NOT NULL, -- "address"."rowid"

        "amount" INT NOT NULL, -- may be negative, if input

        PRIMARY KEY("tx_id", "adr_id"),

        FOREIGN KEY("tx_id") REFERENCES "transaction"("rowid"),
        FOREIGN KEY("adr_id") REFERENCES "address"("rowid")
);`

func InsertTransaction(conn *sqlite.Conn, tx factom.Transaction,
	height uint32, fbOffset int) (txID int64, err error) {
	defer sqlitex.Save(conn)(&err)
	insertTx := conn.Prep(`INSERT INTO "transaction"
                ("height",
                "fb_offset", "size",
                "id", "timestamp",
                "total_fct_in", "total_fct_out", "total_ec_out"
                ) VALUES
                (?, ?, ?, ?, ?, ?, ?, ?);`)
	insertTx.BindInt64(1, int64(height))
	insertTx.BindInt64(2, int64(fbOffset))
	insertTx.BindInt64(3, int64(tx.MarshalBinaryLen()))
	insertTx.BindBytes(4, tx.ID[:])
	insertTx.BindInt64(5, tx.Timestamp.Unix())
	insertTx.BindInt64(6, int64(tx.TotalIn))
	insertTx.BindInt64(7, int64(tx.TotalFCTOut))
	insertTx.BindInt64(8, int64(tx.TotalECOut))

	_, err = insertTx.Step()
	if err != nil {
		return -1, err
	}
	txID = conn.LastInsertRowID()

	insertAdrTx := conn.Prep(`INSERT INTO "address_transaction"
                ("tx_id", "adr_id", "amount") VALUES
                (?, ?, ?)
                ON CONFLICT("tx_id", "adr_id") DO
                UPDATE SET "amount" = "amount" + "excluded"."amount";`)
	insertAdrTx.BindInt64(1, txID)

	sign := int64(-1)
	for _, adrs := range [][]factom.AddressAmount{tx.FCTInputs, tx.FCTOutputs} {
		for _, adr := range adrs {
			amount := sign * int64(adr.Amount)
			adr := adr.FAAddress()
			var adrID int64
			if adrID, err = AddressAdd(conn, &adr, amount); err != nil {
				return -1, err
			}
			insertAdrTx.BindInt64(2, adrID)
			insertAdrTx.BindInt64(3, amount)
			if _, err = insertAdrTx.Step(); err != nil {
				return -1, err
			}
			insertAdrTx.Reset()
		}
		sign = 1
	}

	return txID, nil
}

var selectTransactionWhere = `SELECT "fblock"."rowid", "fb_offset", "size"
        FROM "transaction" JOIN "fblock" ON "transaction"."height" = "fblock"."height" WHERE `

func SelectTransactionByTxID(conn *sqlite.Conn,
	txID *factom.Bytes32) (factom.Transaction, error) {

	stmt := conn.Prep(selectTransactionWhere + `"id" = ?;`)
	stmt.BindBytes(1, txID[:])
	return selectTransaction(conn, stmt)
}
func SelectTransactionByRowID(conn *sqlite.Conn,
	rowID int64) (factom.Transaction, error) {

	stmt := conn.Prep(selectFBlockWhere + `"rowid" = ?;`)
	stmt.BindInt64(1, rowID)
	return selectTransaction(conn, stmt)
}
func selectTransaction(conn *sqlite.Conn, stmt *sqlite.Stmt) (
	factom.Transaction, error) {
	var tx factom.Transaction

	hasRow, err := stmt.Step()
	if err != nil {
		return tx, err
	}
	if !hasRow {
		return tx, fmt.Errorf("no Transaction found")
	}

	fbRowID := stmt.ColumnInt64(0)
	fbOffset := stmt.ColumnInt64(1)
	size := int(stmt.ColumnInt64(2))

	blob, err := conn.OpenBlob("", "fblock", "data", fbRowID, false)
	if err != nil {
		return tx, err
	}

	data := make([]byte, size)
	read, err := blob.ReadAt(data, fbOffset)
	if err != nil {
		return tx, err
	}
	if read != size {
		return tx, fmt.Errorf("unexpected end of Transaction data")
	}

	fmt.Println(data)
	if err := tx.UnmarshalBinary(data); err != nil {
		return tx, fmt.Errorf("factom.Transaction.UnmarshalBinary(): %w", err)
	}

	return tx, nil
}
