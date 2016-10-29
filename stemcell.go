package main

import (
	"fmt"

	"github.com/jhunt/go-db"
	"github.com/starkandwayne/goutils/log"
)

type Stemcell struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	SHA1    string `json:"sha1,omitempty"`
	URL     string `json:"url,omitempty"`
}

func CreateStemcell(d *db.DB, name, url string) error {
	return d.Exec(`INSERT INTO stemcells (name, url) VALUES ($1, $2)`, name, url)
}

func FindAllStemcells(d *db.DB) ([]string, error) {
	l := make([]string, 0)

	r, err := d.Query(`SELECT name FROM stemcells`)
	if err != nil {
		return l, err
	}

	for r.Next() {
		var o string
		if err = r.Scan(&o); err != nil {
			return l, err
		}
		l = append(l, o)
	}

	return l, nil
}

func FindStemcell(d *db.DB, name string) (Stemcell, error) {
	var o Stemcell

	r, err := d.Query(`SELECT name, url FROM stemcells WHERE name = $1`, name)
	if err != nil {
		return o, err
	}

	if !r.Next() {
		return o, fmt.Errorf("stemcell '%s' not found", name)
	}
	if err = r.Scan(&o.Name, &o.URL); err != nil {
		return o, err
	}
	if r.Next() {
		return o, fmt.Errorf("duplicate stemcells found for '%s'", name)
	}

	return o, nil
}

func FindAllStemcellVersions(d *db.DB, name string) ([]Stemcell, error) {
	l := make([]Stemcell, 0)

	r, err := d.Query(`
SELECT
  name,
  version,
  sha1,
  url

FROM stemcell_versions

WHERE name = $1
  AND valid = 1

ORDER BY
  vnum DESC`, name)
	if err != nil {
		return l, err
	}

	for r.Next() {
		var o Stemcell
		if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
			return l, err
		}
		l = append(l, o)
	}

	if len(l) == 0 {
		n, err := d.Count("SELECT * FROM stemcells WHERE name = $1", name)
		if err == nil && n != 0 {
			return l, nil
		}
		return l, fmt.Errorf("stemcell '%s' not found", name)
	}

	return l, nil
}

func FindLatestStemcellVersions(d *db.DB) ([]Stemcell, error) {
	l := make([]Stemcell, 0)

	r, err := d.Query(`
SELECT
  v.name,
  v.version,
  v.sha1,
  v.url

FROM
  stemcell_versions v
  INNER JOIN (
    SELECT
      name,
      MAX(vnum) AS latest

    FROM stemcell_versions
    WHERE valid = 1
    GROUP BY name
  ) q

  ON
        q.name   = v.name
    AND q.latest = v.vnum
`)
	if err != nil {
		return l, err
	}

	for r.Next() {
		var o Stemcell
		if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
			return l, err
		}
		l = append(l, o)
	}

	return l, nil
}

func FindStemcellVersion(d *db.DB, name, version string) (Stemcell, error) {
	var o Stemcell

	where := ""
	args := make([]interface{}, 1)
	args[0] = name

	if version != "" {
		where = "AND version = $2"
		args = append(args, version)
	}

	r, err := d.Query(fmt.Sprintf(`
SELECT
  name,
  version,
  sha1,
  url

FROM
  stemcell_versions

WHERE name = $1
  AND valid = 1
  %s

ORDER BY
  vnum DESC
LIMIT 1
`, where), args...)
	if err != nil {
		return o, err
	}

	if !r.Next() {
		if version != "" {
			return o, fmt.Errorf("version '%s' of stemcell '%s' not found", version, name)
		}
		n, err := d.Count("SELECT * FROM stemcells WHERE name = $1", name)
		if err == nil && n != 0 {
			return o, fmt.Errorf("no known versions for stemcell '%s'", name)
		}
		return o, fmt.Errorf("stemcell '%s' not found", name)
	}
	if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
		return o, err
	}
	if r.Next() {
		return o, fmt.Errorf("duplicate stemcells found for '%s'", name)
	}

	return o, nil
}

func DeleteStemcell(d *db.DB, name string) error {
	err := d.Exec(`DELETE FROM stemcell_versions WHERE name = $1`, name)
	if err != nil {
		return err
	}

	return d.Exec(`DELETE FROM stemcells WHERE name = $1`, name)
}

func DeleteStemcellVersion(d *db.DB, name, version string) error {
	return d.Exec(`DELETE FROM stemcell_versions WHERE name = $1 AND version = $2`, name, version)
}

func CheckStemcellVersion(d *db.DB, name, version string) error {
	stemcell, err := FindStemcell(d, name)
	if err != nil {
		log.Debugf("unable to find stemcell '%s': %s", name, err)
		return err
	}

	/* generate the URL from the template */
	url := urlify(stemcell.URL, version)
	log.Debugf("checking version '%s' of '%s' at '%s'", version, name, url)

	recheck := true
	n, _ := d.Count(`SELECT * FROM stemcell_versions WHERE name = $1 AND version = $2`, name, version)
	if n == 0 {
		recheck = false
		num, err := vnum(version)
		if err != nil {
			return err
		}
		err = d.Exec(`INSERT INTO stemcell_versions (name, version, vnum, valid) VALUES ($1, $2, $3, 0)`,
			name, version, num)
		if err != nil {
			log.Debugf("unable to check version '%s' of '%s': %s", version, name, err)
			return err
		}
	}

	/* do the async part in its own goroutine */
	go func() {
		/* download and SHA1 the file */
		sha1, err := sha1sum(url)
		if err != nil {
			log.Debugf("download/sha1sum failed: %s...", err)
			if !recheck {
				d.Exec(`DELETE FROM stemcell_versions WHERE name = $1 AND version = $2`,
					name, version)
			}
			return
		}

		err = d.Exec(`
	UPDATE stemcell_versions
	SET valid     = 1,
		url       = $3,
		sha1      = $4

	WHERE name    = $1
	  AND version = $2`, name, version, url, sha1)

		if err != nil {
			log.Debugf("unable to check version '%s' of '%s': %s", version, name, err)
			return
		}
	}()

	return nil
}
