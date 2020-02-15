package main

import (
	"flag"
	"os"
	"strings"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/canonical-ledgers/fblock-scan/engine"
)

func parseFlags(cfg *engine.Config) {
	homeDir, _ := os.UserHomeDir()
	flag.StringVar(&cfg.DBURI, "db", homeDir+"/fblock-scan.sqlite3",
		"SQLite Database URI")
	flag.StringVar(&cfg.C.FactomdServer, "s", cfg.C.FactomdServer, "Factomd URL")
	flag.StringVar(&cfg.Price.APIKey, "api-key", "", "CryptoCompare API Key")
	flag.Var((*Whitelist)(&cfg.Whitelist), "whitelist", "Track only these addresses (comma separated list)")
	start := flag.Int64("start-scan", 0, "Start scanning from this height if creating a new database")
	flag.BoolVar(&cfg.Debug, "debug", false, "Print additional debug info")
	flag.BoolVar(&cfg.Speed, "speed", false, "Improve insert speed at the risk of database corruption on crashes")

	flag.Parse()

	cfg.StartScanHeight = uint32(*start)
}

type Whitelist map[factom.FAAddress]struct{}

func (wl Whitelist) String() string {
	if len(wl) == 0 {
		return ""
	}
	var s string
	for adr := range wl {
		s += adr.String() + ","
	}
	return s[:len(s)-1] // omit trailing comma
}

func (wl *Whitelist) Set(adrStr string) error {
	if *wl == nil {
		*wl = make(Whitelist)
	}
	adrStrs := strings.Split(adrStr, ",")
	for _, adrStr := range adrStrs {
		var adr factom.FAAddress
		if err := adr.Set(adrStr); err != nil {
			return err
		}
		(*wl)[adr] = struct{}{}
	}
	return nil
}
