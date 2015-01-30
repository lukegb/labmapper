package admindb

import (
	"database/sql"
	_ "github.com/lib/pq"
	"net/url"

	"log"
)

type AdminDB struct {
	database *sql.DB
}

func Open(server, database, username, password string) (*AdminDB, error) {
	return OpenWithExtra(server, database, username, password, make(map[string]string))
}

func extrasToUrlValues(extras map[string]string) url.Values {
	vals := url.Values{}
	for k := range extras {
		vals[k] = []string{extras[k]}
	}
	return vals
}

// Nope, no credentials here
// but if you know where to look on a lab machine,
// it's easy to find read-only credentials (which is what you need here)
func OpenWithExtra(server, database, username, password string, extras map[string]string) (*AdminDB, error) {
	adb := AdminDB{}

	connUrl := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(username, password),
		Host:     server,
		Path:     "/" + database,
		RawQuery: extrasToUrlValues(extras).Encode(),
	}
	log.Printf("admindb: Connecting to '%s'", connUrl.String())
	db, err := sql.Open("postgres", connUrl.String())
	if err != nil {
		log.Printf("admindb: Failed to connect... %v", err)
		return nil, err
	}
	adb.database = db

	return &adb, nil
}
