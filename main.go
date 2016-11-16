package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jhunt/go-db"
	"github.com/starkandwayne/goutils/log"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "warning"
	}
	log.SetupLogging(log.LogConfig{
		Type:  "console",
		Level: level,
	})
	log.Infof("genesis-index starting up")

	var d *db.DB
	if dsn, err := ParseVcap(os.Getenv("VCAP_SERVICES"), []string{"postgres", "postgresql"}, "uri"); err == nil {
		d, err = Database("postgres", fmt.Sprintf("%s?sslmode=disable", dsn))
		if err != nil {
			log.Infof("Unable to connect to database: %s", err)
			return
		}
	} else if file := os.Getenv("SQLITE_DB"); file != "" {
		d, err = Database("sqlite3", file)
		if err != nil {
			log.Infof("Unable to connect to database: %s", err)
			return
		}
	} else {
		log.Errorf("Unable to determine DSN for backing database")
		log.Errorf("No service tagged 'postgres' is bound (per the VCAP_SERVICES environment variable)")
		log.Errorf("and SQLITE_DB environment variable is not set.")
		return
	}

	/* clean house */
	d.Exec(`DELETE FROM release_versions WHERE valid = 0`)

	/* set up the server */
	mux := http.NewServeMux()
	mux.Handle("/v1/release", ReleaseAPI{db: d})
	mux.Handle("/v1/release/", ReleaseAPI{db: d})
	mux.Handle("/v1/stemcell", StemcellAPI{db: d})
	mux.Handle("/v1/stemcell/", StemcellAPI{db: d})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Infof("listening on *:%s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
}
