package engine

import (
	"context"
	"fmt"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/canonical-ledgers/cryptoprice/v2"
)

type Config struct {
	C               *factom.Client
	DBURI           string
	Whitelist       map[factom.FAAddress]struct{}
	Price           *cryptoprice.Client
	StartScanHeight uint32
}

func NewConfig() Config {
	return Config{C: factom.NewClient(),
		Price: cryptoprice.NewClient("FCT", "USD"),
	}
}

func (cfg Config) String() string {
	s := fmt.Sprintln("factomd:", cfg.C.FactomdServer)
	s += fmt.Sprintln("DB URI:", cfg.DBURI)
	if cfg.Whitelist == nil {
		s += fmt.Sprintln("Tracking All Addresses")
	} else {
		s += fmt.Sprintln("Tracking:")
		for adr := range cfg.Whitelist {
			s += fmt.Sprintln(adr)
		}
	}

	return s
}

func (cfg Config) checkNetworkID(ctx context.Context) error {
	var db factom.DBlock
	if err := db.Get(ctx, cfg.C); err != nil {
		return fmt.Errorf("factom.DBlock.Get(): %w", err)
	}

	if !db.NetworkID.IsMainnet() {
		return fmt.Errorf("connected to Factom %v but expected %v",
			db.NetworkID, factom.MainnetID())
	}
	return nil
}
