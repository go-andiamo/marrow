package marrow

import (
	"database/sql"
	"github.com/go-andiamo/marrow/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNamedTypeDatabases_register(t *testing.T) {
	r := namedDatabases{}
	r.register("mysql", &sql.DB{}, common.DatabaseArgs{})
	require.Len(t, r, 2)
	assert.NotNil(t, r[""])
	assert.NotNil(t, r["mysql"])

	r.register("postgres", &sql.DB{}, common.DatabaseArgs{})
	r.register("mysql", &sql.DB{}, common.DatabaseArgs{})
	require.Len(t, r, 4)
	assert.NotNil(t, r[""])
	assert.NotNil(t, r["mysql"])
	assert.NotNil(t, r["mysql-2"])
	assert.NotNil(t, r["postgres"])
}
