// Command devfixture inserts synthetic, completed games into an existing
// development database. It is not part of the server binary or runtime image.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	path := flag.String("db", "", "existing development SQLite database (required)")
	development := flag.Bool("confirm-development", false, "confirm this is a development database, never production")
	flag.Parse()
	if *path == "" || !*development || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	fixture, err := generate(ctx, *path, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Synthetic development game inserted (4 runners, 16 hunters, 4 hunter teams).\nGame ID: %s\nStart: %s\nEnd: %s\nPositions: %d\nResult path: /results/%s\n",
		fixture.id, fixture.start.Format(time.RFC3339), fixture.end.Format(time.RFC3339), fixture.positions, fixture.id)
}
