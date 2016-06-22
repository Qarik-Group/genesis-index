package main

import (
	"github.com/jhunt/go-db"
)

func Database(driver, dsn string) (*db.DB, error) {
	d := &db.DB{
		Driver: driver,
		DSN:    dsn,
	}

	err := d.Connect()
	if err != nil {
		return nil, err
	}

	s := db.NewSchema()
	s.Version(1, func(d *db.DB) error {
		err := d.Exec(`
  CREATE TABLE releases (
    name  VARCHAR(200)  NOT NULL PRIMARY KEY,
    url   TEXT          NOT NULL
  )
`)
		if err != nil {
			return err
		}

		err = d.Exec(`
  CREATE TABLE stemcells (
    name  VARCHAR(200)  NOT NULL PRIMARY KEY,
    url   TEXT          NOT NULL
  )
`)
		if err != nil {
			return err
		}

		err = d.Exec(`
  CREATE TABLE release_versions (
    name     VARCHAR(200)  NOT NULL,
    version  VARCHAR(20)   NOT NULL,
    sha1     VARCHAR(200)  NOT NULL DEFAULT '',
    url      TEXT          NOT NULL DEFAULT '',
    valid    INTEGER       NOT NULL DEFAULT 0,

    UNIQUE (name, version)
  )
`)
		if err != nil {
			return err
		}

		err = d.Exec(`
  CREATE TABLE stemcell_versions (
    name     VARCHAR(200)  NOT NULL,
    version  VARCHAR(20)   NOT NULL,
    sha1     VARCHAR(200)  NOT NULL DEFAULT '',
    url      TEXT          NOT NULL DEFAULT '',
    valid    INTEGER       NOT NULL DEFAULT 0,

    UNIQUE (name, version)
  )
`)
		if err != nil {
			return err
		}

		return nil
	})

	err = s.Migrate(d, db.Latest)
	if err != nil {
		return nil, err
	}

	return d, nil
}
