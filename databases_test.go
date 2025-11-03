package marrow

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNamedTypeDatabases_register(t *testing.T) {
	r := namedDatabases{}
	r.register("mysql", &sql.DB{}, 0)
	require.Len(t, r, 2)
	assert.NotNil(t, r[""])
	assert.NotNil(t, r["mysql"])

	r.register("postgres", &sql.DB{}, 0)
	r.register("mysql", &sql.DB{}, 0)
	require.Len(t, r, 4)
	assert.NotNil(t, r[""])
	assert.NotNil(t, r["mysql"])
	assert.NotNil(t, r["mysql-2"])
	assert.NotNil(t, r["postgres"])
}
