package main

import (
	"flag"
	"os"

	"github.com/AdamSLevy/fct-trackerd/engine"
)

func parseFlags(cfg *engine.Config) {
	homeDir, _ := os.UserHomeDir()
	flag.StringVar(&cfg.DBURI, "db", homeDir+"/fct-trackerd.sqlite3",
		"SQLite Database URI")
	flag.StringVar(&cfg.C.FactomdServer, "s", cfg.C.FactomdServer, "Factomd URL")
	flag.StringVar(&cfg.Price.APIKey, "api-key", "", "CryptoCompare API Key")
	flag.Parse()
}
