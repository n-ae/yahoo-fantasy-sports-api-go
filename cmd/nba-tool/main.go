// Command nba-tool is the reference application built on the Yahoo Fantasy
// Sports SDK (pkg/yahoo). It bundles the opinionated NBA analytics layer
// (pkg/service, pkg/repository) that is intentionally NOT part of the reusable
// SDK. See docs/v2-roadmap.md: in v2 the service/repository packages move here
// and leave the library's public surface.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/repository"
	"github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/service"
	"github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/yahoo"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "./fantasy.db", "path to the SQLite database")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	client, err := yahoo.NewClientWithOptions(
		yahoo.FromEnv(),           // YAHOO_ACCESS_TOKEN, YAHOO_CONSUMER_KEY, ...
		yahoo.WithSQLiteCache(db), // reuse the app DB for the response cache
	)
	if err != nil {
		log.Fatalf("build Yahoo client: %v", err)
	}

	ctx := context.Background()

	switch args[0] {
	case "import":
		if len(args) < 3 {
			log.Fatal("usage: nba-tool import <yahoo-league-id> <user-team-id>")
		}
		leagueSvc := service.NewLeagueService(
			client,
			repository.NewLeagueRepository(db),
			repository.NewTeamRepository(db),
			repository.NewRosterRepository(db),
			db,
		)
		if err := leagueSvc.ImportLeague(ctx, args[1], args[2]); err != nil {
			log.Fatalf("import league: %v", err)
		}
		fmt.Printf("imported league %s\n", args[1])

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `nba-tool - reference NBA fantasy application for the Yahoo Fantasy SDK

Usage:
  nba-tool [-db path] <command> [args]

Commands:
  import <yahoo-league-id> <user-team-id>   import a league and its rosters

Environment:
  YAHOO_ACCESS_TOKEN, YAHOO_REFRESH_TOKEN, YAHOO_CONSUMER_KEY,
  YAHOO_CONSUMER_SECRET   OAuth configuration (see FromEnv)
`)
}
