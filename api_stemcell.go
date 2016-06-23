package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jhunt/go-db"
	"github.com/starkandwayne/goutils/log"
)

type StemcellAPI struct {
	db *db.DB
}

func (api StemcellAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("RECV: %s %s", r.Method, r.URL.Path)
	switch {
	case match(r, `GET /v1/stemcell`):
		stemcells, err := FindAllStemcells(api.db)
		respond(w, err, 200, stemcells)
		return

	case match(r, `POST /v1/stemcell`):
		if !authed(w, r) {
			return
		}
		var payload struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}

		json.NewDecoder(r.Body).Decode(&payload)
		log.Debugf("creating stemcell '%s' at '%s'", payload.Name, payload.URL)
		err := CreateStemcell(api.db, payload.Name, payload.URL)
		respond(w, err, 200, "success")
		return

	case match(r, `GET /v1/stemcell/[^/]+`):
		name := extract(r, `/v1/stemcell/([^/]+)$`)
		log.Debugf("retrieving all versions of stemcell '%s'", name)
		stemcells, err := FindAllStemcellVersions(api.db, name)
		respond(w, err, 200, stemcells)
		return

	case match(r, `DELETE /v1/stemcell/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/stemcell/([^/]+)`)
		log.Debugf("will stop tracking stemcell '%s'", name)
		err := DeleteStemcell(api.db, name)
		respond(w, err, 200, "deleted")
		return

	case match(r, `GET /v1/stemcell/[^/]+/v/[^/]+`):
		name := extract(r, `/v1/stemcell/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/stemcell/[^/]+/v/([^/]+)`)
		log.Debugf("retrieving version '%s' of stemcell '%s'", vers, name)
		stemcell, err := FindStemcellVersion(api.db, name, vers)
		respond(w, err, 200, stemcell)
		return

	case match(r, `GET /v1/stemcell/[^/]+/metadata`):
		name := extract(r, `/v1/stemcell/([^/]+)/metadata`)
		log.Debugf("retrieving latest version of stemcell '%s'", name)
		stemcell, err := FindStemcell(api.db, name)
		respond(w, err, 200, stemcell)
		return

	case match(r, `GET /v1/stemcell/[^/]+/latest`):
		name := extract(r, `/v1/stemcell/([^/]+)/latest$`)
		log.Debugf("retrieving latest version of stemcell '%s'", name)
		stemcell, err := FindStemcellVersion(api.db, name, "")
		respond(w, err, 200, stemcell)
		return

	case match(r, `PUT /v1/stemcell/[^/]+/v/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/stemcell/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/stemcell/[^/]+/v/([^/]+)`)
		log.Debugf("checking for version '%s' of stemcell '%s'", vers, name)

		go CheckStemcellVersion(api.db, name, vers)
		respond(w, nil, 200, "task started in background")
		return

	case match(r, `DELETE /v1/stemcell/[^/]+/v/[^/]+`):
		if !authed(w, r) {
			return
		}
		name := extract(r, `/v1/stemcell/([^/]+)/v/[^/]+`)
		vers := extract(r, `/v1/stemcell/[^/]+/v/([^/]+)`)
		log.Debugf("dropping version '%s' of stemcell '%s'", vers, name)
		err := DeleteStemcellVersion(api.db, name, vers)
		respond(w, err, 200, fmt.Sprintf("v%s deleted", vers))
		return
	}

	w.WriteHeader(404)
}
