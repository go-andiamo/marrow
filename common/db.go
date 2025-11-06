package common

// DatabaseArgs describes how a sql driver marks args in queries
//
// this information is used when constructing INSERT statements (e.g. marrow.Context .DbInsert)
type DatabaseArgs struct {
	// Style is the style of arg: PositionalDbArgs, NumberedDbArgs or NamedDbArgs
	Style DatabaseArgsStyle
	// Prefix determines the prefix for markers
	//
	// examples:
	//   * "?" (positional) for github.com/go-sql-driver/mysql - gives "?, ?, ..."
	//   * "$" (numbered) for github.com/lib/pq - gives "$1, $2, ..."
	//   * "@p" (numbered) for github.com/denisenkom/go-mssqldb - gives "@p1, @p2, ..."
	//   * ":" (named) for SqlLite - gives ":foo, :bar, ..."
	Prefix string
	// Base is the starting number for NumberedDbArgs - usually 1, but some drivers may start at zero
	Base int
}

// DatabaseArgsStyle is the type of database arg markers that a sql driver uses
//
// Can be one of PositionalDbArgs or NumberedDbArgs
type DatabaseArgsStyle int

const (
	PositionalDbArgs DatabaseArgsStyle = iota // indicates that the sql driver uses positional args (e.g. "github.com/go-sql-driver/mysql" - `?, ?, ?`)
	NumberedDbArgs                            // indicates that the sql driver uses numbered args (e.g. "github.com/lib/pq" - `$1, $2, $3`)
	NamedDbArgs                               // indicates that the sql driver uses named args (e.g. `:foo, :bar`)
)
