package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/Factom-Asset-Tokens/factom"
)

// CreateTableTransaction is the SQL that creates the "transaction" table which
// contains metadata about individual transactions within an FBlock.
const CreateTableTransaction = `CREATE TABLE "transaction" (
        "id"      INTEGER PRIMARY KEY,

        "height" INT NOT NULL,    -- "fblock"."height"

        "fb_offset" INT NOT NULL, -- index of tx data within "fblock"."data"
        "size" INT NOT NULL,      -- length of tx data in bytes

        "timestamp" INT NOT NULL,

        -- amounts
        "total_fct_in"  INT NOT NULL, -- denoted in factoshis
        "total_fct_out" INT NOT NULL, -- denoted in factoshis
        "total_ec_out"  INT NOT NULL, -- denoted in factoshis

        "hash" BLOB NOT NULL, -- hash of tx ledger data

        "memo" TEXT,

        FOREIGN KEY("height") REFERENCES "fblock"("height")
);
`
const CreateIndexTransactionHash = `CREATE INDEX IF NOT EXIST "idx_transaction_id"
        ON "transaction"("hash");`

const CreateTableAddressTransaction = `CREATE TABLE "address_transaction" (
        "tx_id" INT NOT NULL,  -- "transaction"."id"
        "adr_id" INT NOT NULL, -- "address"."id"

        "amount" INT NOT NULL, -- may be negative, if input

        PRIMARY KEY("tx_id", "adr_id"),

        FOREIGN KEY("tx_id") REFERENCES "transaction"("id"),
        FOREIGN KEY("adr_id") REFERENCES "address"("id")
);`

func InsertTransaction(conn *sqlite.Conn, tx factom.Transaction,
	height uint32, fbOffset int) (int64, error) {

	stmt := conn.Prep(`INSERT INTO "transaction" (
                "height",
                "fb_offset",
                "size",
                "hash",
                "timestamp",
                "total_fct_in",
                "total_fct_out",
                "total_ec_out"
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`)
	defer stmt.Reset()

	i := sqlite.BindIncrementor()
	stmt.BindInt64(i(), int64(height))
	stmt.BindInt64(i(), int64(fbOffset))
	stmt.BindInt64(i(), int64(tx.MarshalBinaryLen()))
	stmt.BindBytes(i(), tx.ID[:])
	stmt.BindInt64(i(), tx.Timestamp.Unix())
	stmt.BindInt64(i(), int64(tx.TotalIn))
	stmt.BindInt64(i(), int64(tx.TotalFCTOut))
	stmt.BindInt64(i(), int64(tx.TotalECOut))

	_, err := stmt.Step()
	if err != nil {
		return -1, err
	}

	return conn.LastInsertRowID(), nil
}

var ignoreErr = fmt.Errorf("ignore")

var selectTransactionWhere = `SELECT "height", "fb_offset", "size"
        FROM "transaction" WHERE `

func SelectTransactionByHash(conn *sqlite.Conn,
	txID *factom.Bytes32) (factom.Transaction, error) {

	stmt := conn.Prep(selectTransactionWhere + `"hash" = ?;`)
	defer stmt.Reset()
	stmt.BindBytes(sqlite.BindIndexStart, txID[:])
	return selectTransaction(conn, stmt)
}
func SelectTransactionByID(conn *sqlite.Conn,
	rowID int64) (factom.Transaction, error) {

	stmt := conn.Prep(selectTransactionWhere + `"id" = ?;`)
	defer stmt.Reset()
	stmt.BindInt64(sqlite.BindIndexStart, rowID)
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

	i := sqlite.ColumnIncrementor()
	fblockID := stmt.ColumnInt64(i())
	fbOffset := stmt.ColumnInt64(i())
	size := int(stmt.ColumnInt64(i()))

	blob, err := conn.OpenBlob("", "fblock", "data", fblockID, false)
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

	if err := tx.UnmarshalBinary(data); err != nil {
		return tx, fmt.Errorf("factom.Transaction.UnmarshalBinary(): %w", err)
	}

	return tx, nil
}
