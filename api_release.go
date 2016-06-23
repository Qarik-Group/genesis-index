package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jhunt/go-db"
	"github.com/starkandwayne/goutils/log"
)

type ReleaseAPI struct {
	db *db.DB
}

func (api ReleaseAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("RECV: %s %s", r.Method, r.URL.Path)
	switch {
	case match(r, `GET /v1/release`):
		releases, err := FindAllReleases(api.db)
		respond(w, err, 200, releases)
		return

	case match(r, `POST /v1/release`):
		if !authed(w, r) {
			return
		}
		var payload struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}

		json.NewDecoder(r.Body).Decode(&payload)
		log.Debugf("creating release '%s' at '%s'", payload.Name, payload.URL)
		err := CreateRelease(api.db, payload.Name, payload.URL)
		respond(w, err, 200, "success")
		return

	case match(r, `GET /v1/release/latest`):
		log.Debugf("retrieving latest versions of all releases")
		releases, err := FindLatestReleaseVersions(api.db)
		respond(w, err, 200, releases)
		return

	case match(r, `GET /v1/release/[^/]+`):
		name := extract(r, `/v1/release/([^/]+)$`)
		log.Debugf("retrieving all versions of release '%s'", name)
		releases, err := FindAllReleaseVersions(api.db, name)
		respond(w, err, 200, releases)
		return

	case match(r, `DELETE /v1/release/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/release/([^/]+)`)
		log.Debugf("will stop tracking release '%s'", name)
		err := DeleteRelease(api.db, name)
		respond(w, err, 200, "deleted")
		return

	case match(r, `GET /v1/release/[^/]+/v/[^/]+`):
		name := extract(r, `/v1/release/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/release/[^/]+/v/([^/]+)`)
		log.Debugf("retrieving version '%s' of release '%s'", vers, name)
		release, err := FindReleaseVersion(api.db, name, vers)
		respond(w, err, 200, release)
		return

	case match(r, `GET /v1/release/[^/]+/metadata`):
		name := extract(r, `/v1/release/([^/]+)/metadata`)
		log.Debugf("retrieving latest version of release '%s'", name)
		release, err := FindRelease(api.db, name)
		respond(w, err, 200, release)
		return

	case match(r, `GET /v1/release/[^/]+/latest`):
		name := extract(r, `/v1/release/([^/]+)/latest$`)
		log.Debugf("retrieving latest version of release '%s'", name)
		release, err := FindReleaseVersion(api.db, name, "")
		respond(w, err, 200, release)
		return

	case match(r, `PUT /v1/release/[^/]+/v/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/release/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/release/[^/]+/v/([^/]+)`)
		log.Debugf("checking for version '%s' of release '%s'", vers, name)

		go CheckReleaseVersion(api.db, name, vers)
		respond(w, nil, 200, "task started in background")
		return

	case match(r, `DELETE /v1/release/[^/]+/v/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/release/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/release/[^/]+/v/([^/]+)`)
		log.Debugf("dropping version '%s' of release '%s'", vers, name)
		err := DeleteReleaseVersion(api.db, name, vers)
		respond(w, err, 200, fmt.Sprintf("v%s deleted", vers))
		return
	}

	w.WriteHeader(404)
}
