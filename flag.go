package main

import (
	"flag"
	"os"

	"github.com/canonical-ledgers/fblock-scan/engine"
)

func parseFlags(cfg *engine.Config) {
	homeDir, _ := os.UserHomeDir()
	flag.StringVar(&cfg.DBURI, "db", homeDir+"/fblock-scan.sqlite3",
		"SQLite Database URI")
	flag.StringVar(&cfg.C.FactomdServer, "s", cfg.C.FactomdServer, "Factomd URL")
	flag.StringVar(&cfg.Price.APIKey, "api-key", "", "CryptoCompare API Key")
	flag.Parse()
}
