package db

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/Factom-Asset-Tokens/factom"
)

const CreateTableFBlock = `CREATE TABLE "fblock"(
        "height" INT PRIMARY KEY,
        "key_mr" BLOB NOT NULL UNIQUE,
        "timestamp" INT,
        "data" BLOB,
        "price" REAL -- Denoted in USD
);
`

func InsertFBlock(conn *sqlite.Conn, fb factom.FBlock, price float64) error {
	stmt := conn.Prep(`INSERT INTO "fblock" (
                "height",
                "key_mr",
                "timestamp",
                "data",
                "price") VALUES (?, ?, ?, ?, ?);`)

	stmt.BindInt64(1, int64(fb.Height))
	stmt.BindBytes(2, fb.KeyMR[:])
	stmt.BindInt64(3, fb.Timestamp.Unix())
	stmt.BindFloat(4, price)
	data, err := fb.MarshalBinary()
	if err != nil {
		return fmt.Errorf("factom.FBlock.MarshalBinary(): %w", err)
	}
	stmt.BindBytes(4, data)

	_, err = stmt.Step()
	return err
}

var selectFBlockWhere = `SELECT "data" FROM "fblock" WHERE `

func SelectFBlockByKeyMR(conn *sqlite.Conn, keyMR *factom.Bytes32) (factom.FBlock, error) {
	stmt := conn.Prep(selectFBlockWhere + `"key_mr" = ?;`)
	stmt.BindBytes(1, keyMR[:])
	return selectFBlock(stmt)
}
func SelectFBlockByHeight(conn *sqlite.Conn, height uint32) (factom.FBlock, error) {
	stmt := conn.Prep(selectFBlockWhere + `"height" = ?;`)
	stmt.BindInt64(1, int64(height))
	return selectFBlock(stmt)
}
func selectFBlock(stmt *sqlite.Stmt) (factom.FBlock, error) {
	var fb factom.FBlock

	hasRow, err := stmt.Step()
	if err != nil {
		return fb, err
	}
	if !hasRow {
		return fb, fmt.Errorf("no FBlock found")
	}

	data := make([]byte, stmt.ColumnLen(0))
	stmt.ColumnBytes(0, data)

	if err := fb.UnmarshalBinary(data); err != nil {
		return fb, fmt.Errorf("factom.FBlock.UnmarshalBinary(): %w", err)
	}

	return fb, nil
}
