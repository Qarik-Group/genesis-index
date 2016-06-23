package main

import (
	"fmt"

	"github.com/jhunt/go-db"
	"github.com/starkandwayne/goutils/log"
)

type Release struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	SHA1    string `json:"sha1,omitempty"`
	URL     string `json:"url,omitempty"`
}

func CreateRelease(d *db.DB, name, url string) error {
	return d.Exec(`INSERT INTO releases (name, url) VALUES ($1, $2)`, name, url)
}

func FindAllReleases(d *db.DB) ([]string, error) {
	l := make([]string, 0)

	r, err := d.Query(`SELECT name FROM releases`)
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

func FindRelease(d *db.DB, name string) (Release, error) {
	var o Release

	r, err := d.Query(`SELECT name, url FROM releases WHERE name = $1`, name)
	if err != nil {
		return o, err
	}

	if !r.Next() {
		return o, fmt.Errorf("release '%s' not found", name)
	}
	if err = r.Scan(&o.Name, &o.URL); err != nil {
		return o, err
	}
	if r.Next() {
		return o, fmt.Errorf("duplicate releases found for '%s'", name)
	}

	return o, nil
}

func FindAllReleaseVersions(d *db.DB, name string) ([]Release, error) {
	l := make([]Release, 0)

	r, err := d.Query(`
SELECT
  name,
  version,
  sha1,
  url

FROM release_versions

WHERE name = $1
  AND valid = 1

ORDER BY
  vnum DESC`, name)
	if err != nil {
		return l, err
	}

	for r.Next() {
		var o Release
		if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
			return l, err
		}
		l = append(l, o)
	}

	if len(l) == 0 {
		n, err := d.Count("SELECT * FROM releases WHERE name = $1", name)
		if err == nil && n != 0 {
			return l, nil
		}
		return l, fmt.Errorf("release '%s' not found", name)
	}

	return l, nil
}

func FindLatestReleaseVersions(d *db.DB) ([]Release, error) {
	l := make([]Release, 0)

	r, err := d.Query(`
SELECT
  v.name,
  v.version,
  v.sha1,
  v.url

FROM
  release_versions v
  INNER JOIN (
    SELECT
      name,
      MAX(vnum) AS latest

    FROM release_versions
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
		var o Release
		if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
			return l, err
		}
		l = append(l, o)
	}

	return l, nil
}

func FindReleaseVersion(d *db.DB, name, version string) (Release, error) {
	var o Release

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
  release_versions

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
			return o, fmt.Errorf("version '%s' of release '%s' not found", version, name)
		}
		n, err := d.Count("SELECT * FROM releases WHERE name = $1", name)
		if err == nil && n != 0 {
			return o, fmt.Errorf("no known versions for release '%s'", name)
		}
		return o, fmt.Errorf("release '%s' not found", name)
	}
	if err = r.Scan(&o.Name, &o.Version, &o.SHA1, &o.URL); err != nil {
		return o, err
	}
	if r.Next() {
		return o, fmt.Errorf("duplicate releases found for '%s'", name)
	}

	return o, nil
}

func DeleteRelease(d *db.DB, name string) error {
	err := d.Exec(`DELETE FROM release_versions WHERE name = $1`, name)
	if err != nil {
		return err
	}

	return d.Exec(`DELETE FROM releases WHERE name = $1`, name)
}

func DeleteReleaseVersion(d *db.DB, name, version string) error {
	return d.Exec(`DELETE FROM release_versions WHERE name = $1 AND version = $2`, name, version)
}

func CheckReleaseVersion(d *db.DB, name, version string) {
	release, err := FindRelease(d, name)
	if err != nil {
		log.Debugf("unable to find release '%s': %s", name, err)
		return
	}

	/* generate the URL from the template */
	url := urlify(release.URL, version)
	log.Debugf("checking version '%s' of '%s' at '%s'", version, name, url)

	recheck := true
	n, _ := d.Count(`SELECT * FROM release_versions WHERE name = $1 AND version = $2`, name, version)
	if n == 0 {
		recheck = false
		num, err := vnum(version)
		if err == nil {
			d.Exec(`INSERT INTO release_versions (name, version, vnum, valid) VALUES ($1, $2, $3, 0)`,
				name, version, num)
		}
	}

	/* download and SHA1 the file */
	sha1, err := sha1sum(url)
	if err != nil {
		log.Debugf("download/sha1sum failed: %s...", err)
		if !recheck {
			d.Exec(`DELETE FROM release_versions WHERE name = $1 AND version = $2`,
				name, version)
		}
		return
	}

	err = d.Exec(`
UPDATE release_versions
SET valid     = 1,
    url       = $3,
    sha1      = $4

WHERE name    = $1
  AND version = $2`, name, version, url, sha1)

	if err != nil {
		log.Debugf("unable to check version '%s' of '%s': %s", version, name, err)
		return
	}
}
