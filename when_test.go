package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSetVar(t *testing.T) {
	w := SetVar(After, "foo", "bar")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])
}

func TestClearVars(t *testing.T) {
	w := ClearVars(After)
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(map[Var]any{"foo": "bar"})
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.Empty(t, ctx.vars)
}

func TestDbInsert(t *testing.T) {
	w := DbInsert(After, "table_name", Columns{"foo": "bar"})
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestDbExec(t *testing.T) {
	w := DbExec(After, "DELETE FROM table_name")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestDbClearTable(t *testing.T) {
	w := DbClearTable(After, "table_name")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = w.Run(ctx)
	require.NoError(t, err)
}
