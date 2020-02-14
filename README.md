# fblock-scan

FBlock scan builds a fully normalized SQLite3 database that indexs all FBlocks,
Transaction, and Addresses on the factom blockchain.

## Use
No configuration is required except for a factomd API endpoint.
The public `http://api.factomd.net` can work but may be slow.

```
$ ./fblock-scan -s https://api.factomd.net
fblock-scan: Factoid Block Transaction Scanner
factomd: https://api.factomd.net/v2
DB URI: /home/aslevy/fblock-scan.sqlite3
Tracking All Addresses

Starting...
Engine started.
Scanning from block 0 to 231816...
```
You can stop the scan at any time using CTRL+C.
```
^CSIGINT: Shutting down...
Engine stopped.
```
When you restart it will resume where it left off.

By default the database is stored at `$HOME/fblock-scan.sqlite3`.

### Other flags
```
$ fblock-scan -h
Usage of ./fblock-scan:
  -api-key string
    	CryptoCompare API Key
  -db string
    	SQLite Database URI (default "$HOME/fblock-scan.sqlite3")
  -s string
    	Factomd URL (default "http://localhost:8088/v2")
  -start-scan int
    	Start scanning from this height if creating a new database
  -whitelist value
    	Track only these addresses (comma separated list)
```

Use `-start-scan` to limit the scan to only the earliest blocks that your
addresses of interest were used in.

Use `-whitelist` to only track the balances and transaction metadata of certain
addresses. All transactions will still be indexed by their TxID Hash, but
`address_transaction` relations will only be saved for transactions involving a
whitelisted address. Any other addresses involved in the transaction with a
whitelisted address will be indexed but their balances will be inaccurate.

Use an `-api-key` from [CryptoCompare.com](https://cryptocompare.com) to allow
the program to not be rate limited when querying for FCT prices.

## Schema

Below is the SQLite database schema. FBlock data contains all transaction data,
so to avoid duplicated data, the offset within the FBlock and length of each
transaction is saved. Using the `sqlite3_blob_` interfaces the raw Transaction
data can be efficiently read from the FBlock with the corresponding height.

Price is saved per FBlock.

If an address is tracked, the cumulative input and output amounts of each
transaction associated with that address are saved in the address_transaction
table. Since technically an address can appear more than once in both the
inputs and outputs, only the cumulative amount is saved as a positive (output),
or negative (input) number.

```
CREATE TABLE IF NOT EXISTS "fblock"(
        "height" INT PRIMARY KEY,
        "key_mr" BLOB NOT NULL UNIQUE,
        "timestamp" INT,
        "data" BLOB,
        "price" REAL -- Denoted in USD
);
CREATE TABLE IF NOT EXISTS "address" (
        "address" TEXT PRIMARY KEY NOT NULL,
        "balance" INTEGER NOT NULL,
        "memo"    TEXT
);
CREATE TABLE IF NOT EXISTS "transaction" (
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
CREATE TABLE IF NOT EXISTS "address_transaction" (
        "tx_id" INT NOT NULL,  -- "transaction"."rowid"
        "adr_id" INT NOT NULL, -- "address"."rowid"

        "amount" INT NOT NULL, -- may be negative, if input

        PRIMARY KEY("tx_id", "adr_id"),

        FOREIGN KEY("tx_id") REFERENCES "transaction"("rowid"),
        FOREIGN KEY("adr_id") REFERENCES "address"("rowid")
);
```
