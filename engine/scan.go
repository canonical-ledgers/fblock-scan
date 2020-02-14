package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"crawshaw.io/sqlite"
	"github.com/AdamSLevy/retry"
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/canonical-ledgers/fblock-scan/db"
	"github.com/cheggaaa/pb/v3"
)

func (cfg Config) Start(ctx context.Context) (_ <-chan struct{}, err error) {

	if err := cfg.checkNetworkID(ctx); err != nil {
		return nil, err
	}

	conn, err := sqlite.OpenConn(cfg.DBURI, 0)
	if err != nil {
		return nil, err
	}
	conn.SetInterrupt(ctx.Done())

	err = db.Setup(conn)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()
		if err := cfg.scan(ctx, conn); err != nil {
			if ctx.Err() == nil {
				log.Println("Error: ", err)
			}
		}
	}()
	return done, nil
}
func (cfg Config) scan(ctx context.Context, conn *sqlite.Conn) error {

	// synced tracks whether we have completed our first sync.
	var synced bool

	syncHeight, err := db.SelectSyncHeight(conn)
	if err != nil {
		return err
	}
	if syncHeight > 0 {
		syncHeight++
	} else {
		syncHeight = cfg.StartScanHeight
	}

	var heights factom.Heights
	if err := heights.Get(ctx, cfg.C); err != nil {
		return fmt.Errorf("factom.Heights.Get(): %v", err)
	}

	syncBar := pb.New(int(heights.EntryBlock))
	syncBar.Add(int(int32(syncHeight - 1)))
	syncBar.Start()

	// scanTicker kicks off a new scan.
	scanTicker := time.NewTicker(5 * time.Minute)

	defer fmt.Println()
	// Factom Blockchain Scan Loop
	for {
		if !synced && syncHeight == heights.EntryBlock {
			synced = true
			syncBar.Finish()
			fmt.Printf("FBlock scan complete to block %v.", syncHeight)
		}
		// Process all new DBlocks sequentially.
		for h := syncHeight; h <= heights.EntryBlock; h++ {
			syncBar.Increment()
			if err := cfg.syncFBlock(ctx, conn, h); err != nil {
				return err
			}
			select {
			case <-scanTicker.C:
				err := heights.Get(ctx, cfg.C)
				if err != nil {
					return fmt.Errorf("factom.Heights.Get(): %v", err)
				}
				syncBar.SetTotal(int64(heights.EntryBlock))
			default:
			}
		}

		if synced {
			// Wait until the next scan tick or we're told to stop.
			select {
			case <-scanTicker.C:
			case <-ctx.Done():
				return nil
			}
		}

		// Check the Factom blockchain height but log and retry if this
		// request fails.
		err := heights.Get(ctx, cfg.C)
		if err != nil {
			return fmt.Errorf("factom.Heights.Get(): %v", err)
		}
	}
}
func (cfg Config) syncFBlock(ctx context.Context, conn *sqlite.Conn,
	height uint32) error {

	dblk := factom.DBlock{Height: height}
	if err := dblk.Get(ctx, cfg.C); err != nil {
		return err
	}

	policy := retry.LimitTotal{Limit: 30 * time.Minute,
		Policy: retry.LimitAttempts{Limit: 200,
			Policy: retry.Max{Cap: 2 * time.Minute,
				Policy: retry.Randomize{Factor: .25,
					Policy: retry.Exponential{
						Initial:    2 * time.Second,
						Multiplier: 1.3}}}}}

	var price float64
	retry.Run(ctx, policy, nil,
		func(err error, n uint, next time.Duration) {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("%v attempts, next in %v\n", n, next)
		},
		func() (err error) {
			// Get price at Timestamp
			price, err = cfg.Price.GetPriceAt(dblk.Timestamp)
			if err != nil {
				return fmt.Errorf("cryptoprice.Client.GetPriceAt(): %w",
					err)
			}
			return nil
		})

	fb := dblk.FBlock
	if err := fb.Get(ctx, cfg.C); err != nil {
		return err
	}

	if err := db.InsertFBlock(conn, fb, price, cfg.Whitelist); err != nil {
		return fmt.Errorf("db.InsertFBlock(): %w", err)
	}

	return nil
}
