package engine

import (
	"context"
	"log"

	"crawshaw.io/sqlite"
	"github.com/AdamSLevy/fct-trackerd/db"
)

func Start(ctx context.Context, dbURI string) (_ <-chan struct{}, err error) {

	conn, err := sqlite.OpenConn(dbURI, 0)
	if err != nil {
		return nil, err
	}

	err = db.Setup(conn)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()
		if err := run(ctx, conn); err != nil {
			log.Println("Error: ", err)
		}
	}()
	return done, nil
}

func run(ctx context.Context, conn *sqlite.Conn) error {
	<-ctx.Done()
	return nil
}
