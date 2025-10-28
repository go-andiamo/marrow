package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func Test_setVar(t *testing.T) {
	c := &setVar{
		name:  "foo",
		value: "bar",
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "foo", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])
}

func Test_clearVars(t *testing.T) {
	c := &clearVars{
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "CLEAR VARS", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(map[Var]any{"foo": "bar"})
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Empty(t, ctx.vars)
}

func Test_dbInsert(t *testing.T) {
	c := &dbInsert{
		tableName: "table",
		row:       Columns{"foo": "bar"},
		frame:     framing.NewFrame(0),
	}
	assert.Equal(t, "INSERT table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = c.Run(ctx)
	require.NoError(t, err)
}

func Test_dbExec(t *testing.T) {
	c := &dbExec{
		query: "DELETE FROM table",
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "EXEC DELETE FROM table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = c.Run(ctx)
	require.NoError(t, err)
}

func Test_dbClearTable(t *testing.T) {
	c := &dbClearTable{
		tableName: "table",
		frame:     framing.NewFrame(0),
	}
	assert.Equal(t, "DELETE FROM table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.db = db
	err = c.Run(ctx)
	require.NoError(t, err)
}

func Test_userDefinedCapture(t *testing.T) {
	c := &userDefinedCapture{
		fn: func(ctx Context) error {
			ctx.SetVar("foo", "bar")
			return nil
		},
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "(User Defined PreCapture)", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])

	c.name = "set foo"
	assert.Equal(t, "set foo", c.Name())
}

func Test_setCookie(t *testing.T) {
	c := &setCookie{
		cookie: &http.Cookie{Name: "foo", Value: "bar"},
		frame:  framing.NewFrame(0),
	}
	assert.Equal(t, "SET COOKIE foo", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.cookieJar, 1)
}

func Test_storeCookie(t *testing.T) {
	c := &storeCookie{
		name:  "foo",
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "STORE COOKIE foo", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	err := c.Run(ctx)
	require.Error(t, err)

	res := &http.Response{
		Header: http.Header{},
	}
	cookie := &http.Cookie{Name: "foo", Value: "bar"}
	res.Header.Add("Set-Cookie", cookie.String())
	ctx.currResponse = res
	err = c.Run(ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.cookieJar, 1)

	c.name = "no such cookie"
	err = c.Run(ctx)
	require.Error(t, err)
}
