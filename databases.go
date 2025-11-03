package marrow

import (
	"database/sql"
	"github.com/go-andiamo/marrow/common"
	"strconv"
)

type namedDatabases map[string]*namedDb

type namedDb struct {
	db         *sql.DB
	argMarkers common.DatabaseArgMarkers
}

func (nd namedDatabases) register(name string, db *sql.DB, argMarker common.DatabaseArgMarkers) {
	if db != nil {
		tdb := &namedDb{db: db, argMarkers: argMarker}
		if len(nd) == 0 {
			nd[""] = tdb
			nd[name] = tdb
		} else if _, ok := nd[name]; ok {
			for idx := 2; ; idx++ {
				k := name + "-" + strconv.Itoa(idx)
				if _, exists := nd[k]; !exists {
					nd[k] = tdb
					break
				}
			}
		} else {
			nd[name] = tdb
		}
	}
}
