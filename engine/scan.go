package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/AdamSLevy/retry"
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/canonical-ledgers/fblock-scan/db"
	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/errgroup"
)

func (cfg Config) Start(ctx context.Context) (_ <-chan struct{}, err error) {

	if err := cfg.checkNetworkID(ctx); err != nil {
		return nil, err
	}

	conn, err := sqlite.OpenConn(cfg.DBURI, 0)
	if err != nil {
		return nil, err
	}
	g, ctx := errgroup.WithContext(ctx)
	conn.SetInterrupt(ctx.Done())

	err = db.Setup(conn, cfg.Speed)
	if err != nil {
		return nil, err
	}

	syncHeight, err := db.SelectSyncHeight(conn)
	if err != nil {
		return nil, err
	}
	if syncHeight > 0 {
		syncHeight++
	} else {
		syncHeight = cfg.StartScanHeight
	}

	cfg.syncBar = pb.New(0)

	fblocks := make(chan fbPrice, 20)
	g.Go(func() error { return cfg.fblockInserter(ctx, conn, fblocks) })
	g.Go(func() error { return cfg.scan(ctx, syncHeight, fblocks) })

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()
		if err := g.Wait(); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Println("Error: ", err)
			}
		}
	}()
	return done, nil
}
func (cfg Config) scan(ctx context.Context, syncHeight uint32,
	fblocks chan<- fbPrice) error {

	// synced tracks whether we have completed our first sync.
	var synced bool

	var heights factom.Heights
	if err := heights.Get(ctx, cfg.C); err != nil {
		return fmt.Errorf("factom.Heights.Get(): %v", err)
	}

	cfg.syncBar.SetTotal(int64(heights.EntryBlock))
	cfg.syncBar.Add(int(int32(syncHeight - 1)))
	cfg.syncBar.Start()

	// scanTicker kicks off a new scan.
	scanTicker := time.NewTicker(5 * time.Minute)

	defer fmt.Println()
	// Factom Blockchain Scan Loop
	for {
		if !synced && syncHeight == heights.EntryBlock {
			synced = true
			cfg.syncBar.Finish()
			fmt.Printf("FBlock scan complete to block %v.", syncHeight)
		}
		// Process all new DBlocks sequentially.
		for ; syncHeight <= heights.EntryBlock; syncHeight++ {
			if err := cfg.syncFBlock(ctx, syncHeight, fblocks); err != nil {
				return err
			}
			select {
			case <-scanTicker.C:
				err := heights.Get(ctx, cfg.C)
				if err != nil {
					return fmt.Errorf(
						"factom.Heights.Get(): %v", err)
				}
				cfg.syncBar.SetTotal(int64(heights.EntryBlock))
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
func (cfg Config) syncFBlock(ctx context.Context, height uint32,
	fblocks chan<- fbPrice) error {

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
				return fmt.Errorf(
					"cryptoprice.Client.GetPriceAt(): %w", err)
			}
			return nil
		})

	fb := dblk.FBlock
	if err := fb.Get(ctx, cfg.C); err != nil {
		return err
	}

	fblocks <- fbPrice{fb, price}

	return nil
}

type fbPrice struct {
	factom.FBlock
	Price float64
}

func (cfg Config) fblockInserter(ctx context.Context, conn *sqlite.Conn,
	fblocks <-chan fbPrice) error {
	conn.SetInterrupt(nil)
	for {
		// Batch FBlocks in transactions of 100 for improved
		// performance.
		var commit error
		release := sqlitex.Save(conn)
		for i := 0; i < 100; i++ {
			select {
			case fbp := <-fblocks:
				if err := db.InsertFBlock(conn, fbp.FBlock, fbp.Price,
					cfg.Whitelist); err != nil {
					release(&commit)
					return fmt.Errorf("db.InsertFBlock(): %w", err)
				}
			case <-ctx.Done():
				release(&commit)
				return ctx.Err()
			}
			cfg.syncBar.Increment()

			if cfg.syncBar.Current() == cfg.syncBar.Total() {
				// Generate indexes after sync.
				err := sqlitex.ExecScript(conn,
					db.CreateIndexFBlockKeyMR+
						db.CreateIndexTransactionHash)
				if err != nil {
					release(&commit)
					return err
				}
			}
		}
		release(&commit)
	}
}
