package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/starkandwayne/goutils/log"
)

func urlify(template string, version string) string {
	re := regexp.MustCompile("{{version}}")
	return re.ReplaceAllLiteralString(template, version)
}

func sha1sum(url string) (string, error) {
	r, err := http.Get(url)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	io.Copy(h, r.Body)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func match(req *http.Request, pattern string) bool {
	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func extract(r *http.Request, pattern string) string {
	re := regexp.MustCompile(fmt.Sprintf("^%s$", pattern))
	return re.FindStringSubmatch(r.URL.Path)[1]
}

func bail(w http.ResponseWriter, e error) {
	w.WriteHeader(500)

	fmt.Printf("responding with an error: [%s]\n", e)
	x := struct {
		E string `json:"e"`
	}{E: e.Error()}
	b, err := json.Marshal(x)
	if err == nil {
		fmt.Fprintf(w, "%s\n", string(b))
	} else {
		fmt.Fprintf(w, `{"e":"failed to prepare JSON response"}%s`, "\n")
	}
}

func respond(w http.ResponseWriter, e error, status int, payload interface{}) {
	w.Header().Set("Content-type", "application/json")

	if e != nil {
		bail(w, e)
		return
	}

	if payload != nil {
		if s, ok := payload.(string); ok {
			fmt.Printf("SEND %d %s\n", status, s)
			payload = struct {
				M string `json:"m"`
			}{M: s}
		}

		b, err := json.Marshal(payload)
		if err == nil {
			w.WriteHeader(status)
			fmt.Fprintf(w, "%s\n", string(b))
		} else {
			bail(w, err)
		}
	}
	return
}

func authed(w http.ResponseWriter, r *http.Request) bool {
	auth_user := os.Getenv("AUTH_USERNAME")
	auth_pass := os.Getenv("AUTH_PASSWORD")

	if auth_user == "" {
		log.Debugf("no AUTH_USERNAME set in environment; skipping auth checks")
		return true
	}

	try_user, try_pass, provided := r.BasicAuth()
	if !provided {
		log.Debugf("no Authorization header provided.  returning a 401")
		w.WriteHeader(401)
		return false
	}

	if try_user == auth_user && try_pass == auth_pass {
		return true
	}

	log.Debugf("authorization failed for user '%s'", try_user)
	w.WriteHeader(403)
	return false
}

func vnum(v string) (uint64, error) {
	sem := strings.Split(v, ".")
	for len(sem) < 3 {
		sem = append(sem, "0")
	}

	var n uint64 = 0
	for i := 0; i < 3; i++ {
		u, err := strconv.ParseUint(sem[i], 10, 64)
		if err != nil {
			log.Debugf("vnum had an issue with '%s': %s", v, err)
			return n, err
		}
		n = n*1000000 + u
	}

	return n, nil
}
