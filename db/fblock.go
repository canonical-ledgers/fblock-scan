package db

import (
	"fmt"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/Factom-Asset-Tokens/factom"
)

const CreateTableFBlock = `CREATE TABLE "fblock"(
        "height" INT PRIMARY KEY,
        "timestamp" INT NOT NULL,
        "tx_count" INT NOT NULL,
        "ec_exchange_rate" INT NOT NULL,
        "price" REAL, -- Denoted in USD
        "key_mr" BLOB NOT NULL UNIQUE,
        "data" BLOB NOT NULL
);
`

func InsertFBlock(conn *sqlite.Conn, fb factom.FBlock, price float64,
	whitelist map[factom.FAAddress]struct{}) (err error) {

	if err = checkFBlockContinuity(conn, fb); err != nil {
		return err
	}

	data, err := fb.MarshalBinary()
	if err != nil {
		return fmt.Errorf("factom.FBlock.MarshalBinary(): %w", err)
	}

	defer sqlitex.Save(conn)(&err)

	stmt := conn.Prep(`INSERT INTO "fblock" (
                "height",
                "timestamp",
                "tx_count",
                "ec_exchange_rate",
                "price",
                "key_mr",
                "data"
        ) VALUES (?, ?, ?, ?, ?, ?, ?);`)
	defer stmt.Reset()

	i := sqlite.BindIncrementor()
	stmt.BindInt64(i(), int64(fb.Height))
	stmt.BindInt64(i(), fb.Timestamp.Unix())
	stmt.BindInt64(i(), int64(len(fb.Transactions)))
	stmt.BindInt64(i(), int64(fb.ECExchangeRate))
	if price > 0 {
		stmt.BindFloat(i(), price)
	} else {
		stmt.BindNull(i())
	}
	stmt.BindBytes(i(), fb.KeyMR[:])
	stmt.BindBytes(i(), data)

	_, err = stmt.Step()
	if err != nil {
		return err
	}

	return InsertAllTransactions(conn, fb, whitelist)
}
func checkFBlockContinuity(conn *sqlite.Conn, fb factom.FBlock) error {
	if fb.PrevKeyMR.IsZero() {
		// This is the first FBlock in the chain.
		return nil
	}
	prevKeyMR, err := SelectFBlockKeyMR(conn, fb.Height-1)
	if err != nil {
		return fmt.Errorf("fblock.SelectFBlockKeyMR(height: %v): %w",
			fb.Height-1, err)
	}
	if *fb.PrevKeyMR != prevKeyMR {
		return fmt.Errorf("invalid FBlock.PrevKeyMR, expected:%v but got:%v",
			prevKeyMR, fb.PrevKeyMR)
	}
	return nil
}

func InsertAllTransactions(conn *sqlite.Conn, fb factom.FBlock,
	whitelist map[factom.FAAddress]struct{}) (err error) {
	defer sqlitex.Save(conn)(&err)
	offset := factom.FBlockHeaderMinSize + len(fb.Expansion)
	lastTs := fb.Timestamp
	for _, tx := range fb.Transactions {
		// Advance the offset past any minute markers.
		offset += int(tx.Timestamp.Sub(lastTs) / time.Minute)

		txID, err := InsertTransaction(conn, tx, fb.Height, offset)
		if err != nil {
			return err
		}

		if err := InsertAddresses(conn, tx, txID, whitelist); err != nil {
			return err
		}

		// Advance the offset to the next tx.
		offset += tx.MarshalBinaryLen()
		lastTs = tx.Timestamp
	}
	return nil
}

var selectFBlockWhere = `SELECT "data" FROM "fblock" WHERE `

func SelectFBlockByKeyMR(conn *sqlite.Conn, keyMR *factom.Bytes32) (factom.FBlock, error) {
	stmt := conn.Prep(selectFBlockWhere + `"key_mr" = ?;`)
	defer stmt.Reset()
	stmt.BindBytes(sqlite.BindIndexStart, keyMR[:])
	return selectFBlock(stmt)
}
func SelectFBlockByHeight(conn *sqlite.Conn, height uint32) (factom.FBlock, error) {
	stmt := conn.Prep(selectFBlockWhere + `"height" = ?;`)
	defer stmt.Reset()
	stmt.BindInt64(sqlite.BindIndexStart, int64(height))
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

	i := sqlite.ColumnIndexStart
	data := make([]byte, stmt.ColumnLen(i))
	stmt.ColumnBytes(i, data)

	if err := fb.UnmarshalBinary(data); err != nil {
		return fb, fmt.Errorf("factom.FBlock.UnmarshalBinary(): %w", err)
	}

	return fb, nil
}

func SelectSyncHeight(conn *sqlite.Conn) (uint32, error) {
	stmt := conn.Prep(`SELECT "height" FROM "fblock" ORDER BY "height" DESC LIMIT 1;`)
	defer stmt.Reset()
	hasRow, err := stmt.Step()
	if err != nil || !hasRow {
		return 0, err
	}
	height := uint32(stmt.ColumnInt64(sqlite.ColumnIndexStart))
	return height, nil
}

// SelectFBlockKeyMR returns the KeyMR for the FBlock with sequence seq.
func SelectFBlockKeyMR(conn *sqlite.Conn, height uint32) (factom.Bytes32, error) {
	var keyMR factom.Bytes32
	stmt := conn.Prep(`SELECT "key_mr" FROM "fblock" WHERE "height" = ?;`)
	defer stmt.Reset()
	stmt.BindInt64(sqlite.BindIndexStart, int64(int32(height)))

	hasRow, err := stmt.Step()
	if err != nil {
		return keyMR, err
	}
	if !hasRow {
		return keyMR, fmt.Errorf("no FBlock found")
	}

	if stmt.ColumnBytes(sqlite.ColumnIndexStart, keyMR[:]) != len(keyMR) {
		return keyMR, fmt.Errorf("invalid key_mr length")
	}

	return keyMR, err
}
