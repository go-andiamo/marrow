package common

// DatabaseArgMarkers is the type of database arg markers that a sql driver uses
//
// Can be one of PositionalDbArgs or NumberedDbArgs
type DatabaseArgMarkers int

const (
	PositionalDbArgs DatabaseArgMarkers = iota // indicates that the sql driver uses positional args (e.g. "github.com/go-sql-driver/mysql" - `?, ?, ?`)
	NumberedDbArgs                             // indicates that the sql driver uses numbered args (e.g. "github.com/lib/pq" - `$1, $2, $3`)
)
