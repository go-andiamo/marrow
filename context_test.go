package marrow

import (
	"bytes"
	context2 "context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/mocks/service"
	htesting "github.com/go-andiamo/marrow/testing"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewContext(t *testing.T) {
	ctx := newContext()
	require.NotNil(t, ctx)
	assert.NotNil(t, ctx.coverage)
	assert.NotNil(t, ctx.vars)
	assert.NotNil(t, ctx.cookieJar)
	assert.Equal(t, "", ctx.Host())
}

func TestContext_Vars(t *testing.T) {
	ctx := newContext()
	vars := ctx.Vars()
	assert.Empty(t, vars)
	// ensure cloned
	vars["foo"] = "bar"
	assert.Empty(t, ctx.Vars())
}

func TestContext_SetVar(t *testing.T) {
	ctx := newContext()
	assert.Empty(t, ctx.Vars())
	ctx.SetVar("foo", "bar")
	assert.NotEmpty(t, ctx.Vars())
	vars := ctx.Vars()
	assert.Equal(t, "bar", vars["foo"])
}

func TestContext_ClearVars(t *testing.T) {
	ctx := newContext()
	assert.Empty(t, ctx.Vars())
	ctx.SetVar("foo", "bar")
	assert.NotEmpty(t, ctx.Vars())
	ctx.ClearVars()
	assert.Empty(t, ctx.Vars())
}

func TestContext_Ctx(t *testing.T) {
	ctx := newContext()
	gctx := ctx.Ctx()
	assert.Equal(t, context2.Background(), gctx)
}

func TestContext_Cookies(t *testing.T) {
	ctx := newContext()
	c := ctx.GetCookie("session")
	assert.Nil(t, c)
	ctx.StoreCookie(&http.Cookie{Name: "session", Value: "foo"})
	c = ctx.GetCookie("session")
	assert.NotNil(t, c)
}

func TestContext_GetMockService(t *testing.T) {
	ctx := newContext()
	ctx.mockServices["foo"] = &mockService{}
	m := ctx.GetMockService("foo")
	assert.NotNil(t, m)
}

func TestContext_GetImage(t *testing.T) {
	ctx := newContext()
	ctx.images["foo"] = &mockImage{}
	i := ctx.GetImage("foo")
	assert.NotNil(t, i)
}

func TestContext_GetApiImage(t *testing.T) {
	ctx := newContext()
	ctx.apiImage = &mockApiImage{}
	i := ctx.GetApiImage()
	assert.NotNil(t, i)
}

func TestContext_Db(t *testing.T) {
	ctx := newContext()
	assert.Nil(t, ctx.Db(""))
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx.dbs.register("", db, common.DatabaseArgs{})
	assert.NotNil(t, ctx.Db(""))
}

func TestContext_DbInsert(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		ctx.SetVar("id", 1)
		mock.ExpectExec(`INSERT INTO table \(id\) VALUES \(\?\)`).WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbInsert("", "table", Columns{
			"id": Var("id"),
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("basic, NumberedDbArgs", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{Style: common.NumberedDbArgs, Base: 1, Prefix: "@"})

		ctx.SetVar("id", 1)
		mock.ExpectExec(`INSERT INTO table \(id\) VALUES \(\@1\)`).WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbInsert("", "table", Columns{
			"id": Var("id"),
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("basic, NamedDbArgs", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{Style: common.NamedDbArgs, Prefix: "@"})

		ctx.SetVar("id", 1)
		mock.ExpectExec(`INSERT INTO table \(id\) VALUES \(\@id\)`).WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbInsert("", "table", Columns{
			"id": Var("id"),
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("raw query column", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		ctx.SetVar("id", 1)
		mock.ExpectExec(`INSERT INTO table \(id\) VALUES \(\(SELECT id FROM other WHERE id = 1\)\)`).WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbInsert("", "table", Columns{
			"id": RawQuery("SELECT id FROM other WHERE id = {$id}"),
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("raw query column, missing var", func(t *testing.T) {
		ctx := newContext()
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		err = ctx.DbInsert("", "table", Columns{
			"id": RawQuery("SELECT id FROM other WHERE id = {$id}"),
		})
		require.Error(t, err)
		assert.Equal(t, "unresolved variables in string \"SELECT id FROM other WHERE id = {$id}\"", err.Error())
	})
	t.Run("json column value", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		mock.ExpectExec(``).WithArgs(`{"foo":"bar"}`).WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbInsert("", "table", Columns{
			"id": map[string]any{"foo": "bar"},
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("db error", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		mock.ExpectExec(``).WillReturnError(errors.New("fooey"))

		err = ctx.DbInsert("", "table", Columns{
			"id": 1,
		})
		require.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("missing var", func(t *testing.T) {
		ctx := newContext()
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		err = ctx.DbInsert("", "table", Columns{
			"id": Var("id"),
		})
		require.Error(t, err)
		assert.Equal(t, "unknown variable \"id\"", err.Error())
	})
	t.Run("no db name", func(t *testing.T) {
		ctx := newContext()
		err := ctx.DbInsert("foo", "table", Columns{})
		require.Error(t, err)
		assert.Equal(t, "db name \"foo\" not found", err.Error())
	})
}

func TestContext_DbExec(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		ctx.SetVar("id", "foo")
		mock.ExpectExec("DELETE FROM table WHERE id = ?").WithArgs("foo").WillReturnResult(sqlmock.NewResult(1, 1))

		err = ctx.DbExec("", "DELETE FROM table WHERE id = ?", Var("id"))
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("db error", func(t *testing.T) {
		ctx := newContext()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		ctx.SetVar("id", "foo")
		mock.ExpectExec("DELETE FROM table").WillReturnError(errors.New("fooey"))

		err = ctx.DbExec("", "DELETE FROM table")
		require.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("missing var", func(t *testing.T) {
		ctx := newContext()
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		ctx.dbs.register("", db, common.DatabaseArgs{})

		err = ctx.DbExec("", "DELETE FROM table WHERE id = ?", Var("id"))
		require.Error(t, err)
		assert.Equal(t, "unknown variable \"id\"", err.Error())
	})
	t.Run("no db name", func(t *testing.T) {
		ctx := newContext()
		err := ctx.DbExec("foo", "")
		require.Error(t, err)
		assert.Equal(t, "db name \"foo\" not found", err.Error())
	})
}

func TestContext_Currents(t *testing.T) {
	t.Run("initial empty", func(t *testing.T) {
		ctx := newContext()
		assert.Nil(t, ctx.CurrentEndpoint())
		assert.Equal(t, "", ctx.CurrentUrl())
		assert.Nil(t, ctx.CurrentMethod())
		assert.Nil(t, ctx.CurrentRequest())
		assert.Nil(t, ctx.CurrentResponse())
		assert.Nil(t, ctx.CurrentBody())
	})
	t.Run("endpoint clears", func(t *testing.T) {
		ctx := newContext()
		ctx.currMethod = Method(GET, "")
		ctx.currRequest = httptest.NewRequest("GET", "/", nil)
		ctx.currResponse = &http.Response{}
		ctx.currBody = true
		ctx.setCurrentEndpoint(Endpoint("/foos", ""))
		assert.NotNil(t, ctx.CurrentEndpoint())
		assert.Equal(t, "/foos", ctx.CurrentUrl())
		assert.Nil(t, ctx.CurrentMethod())
		assert.Nil(t, ctx.CurrentRequest())
		assert.Nil(t, ctx.CurrentResponse())
		assert.Nil(t, ctx.CurrentBody())
	})
	t.Run("method clears", func(t *testing.T) {
		ctx := newContext()
		ctx.currEndpoint = Endpoint("/foos", "")
		ctx.currRequest = httptest.NewRequest("GET", "/", nil)
		ctx.currResponse = &http.Response{}
		ctx.currBody = true
		ctx.setCurrentMethod(Method(GET, ""))
		assert.NotNil(t, ctx.CurrentEndpoint())
		assert.NotNil(t, ctx.CurrentMethod())
		assert.Nil(t, ctx.CurrentRequest())
		assert.Nil(t, ctx.CurrentResponse())
		assert.Nil(t, ctx.CurrentBody())
	})
	t.Run("body", func(t *testing.T) {
		ctx := newContext()
		assert.Nil(t, ctx.CurrentBody())
		ctx.setCurrentBody(true)
		assert.NotNil(t, ctx.CurrentBody())
	})
}

func TestContext_doRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := newContext()
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ctx.setCurrentRequest(httptest.NewRequest(http.MethodGet, "/", nil))
		res, ok := ctx.doRequest()
		require.True(t, ok)
		assert.Len(t, cov.Timings, 1)
		assert.Nil(t, cov.Timings[0].Trace)
		require.NotNil(t, res)
		currReq := ctx.CurrentRequest()
		require.NotNil(t, currReq)
		currRes := ctx.CurrentResponse()
		require.NotNil(t, currRes)
		assert.Equal(t, http.StatusOK, currRes.StatusCode)
		body, err := io.ReadAll(currRes.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"foo":"bar"}`, string(body))
	})
	t.Run("with traceTimings", func(t *testing.T) {
		ctx := newContext()
		ctx.traceTimings = true
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ctx.setCurrentRequest(httptest.NewRequest(http.MethodGet, "/", nil))
		_, ok := ctx.doRequest()
		require.True(t, ok)
		assert.Len(t, cov.Timings, 1)
		assert.NotNil(t, cov.Timings[0].Trace)
	})
	t.Run("failure", func(t *testing.T) {
		ctx := newContext()
		ctx.httpDo = &dummyDo{err: errors.New("fooey")}
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ctx.setCurrentRequest(httptest.NewRequest(http.MethodGet, "/", nil))
		_, ok := ctx.doRequest()
		require.False(t, ok)
		assert.Len(t, cov.Failures, 1)
		assert.Equal(t, `fooey`, cov.Failures[0].Error.Error())
	})
}

func TestContext_reportFailure(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ctx.reportFailure(errors.New("fooey"))
		assert.Len(t, cov.Failures, 1)
		assert.Equal(t, `fooey`, cov.Failures[0].Error.Error())
	})
	t.Run("with testing", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		ctx.reportFailure(errors.New("fooey"))
		assert.Len(t, cov.Failures, 1)
		assert.Equal(t, `fooey`, cov.Failures[0].Error.Error())
		assert.Contains(t, buf.String(), ": fooey")
		assert.True(t, ctx.testing.Failed())
	})
	t.Run("with testing & Error", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		capture := &setVar{
			name:  "cap",
			value: "foo",
			frame: framing.NewFrame(0),
		}
		ctx.reportFailure(wrapCaptureError(errors.New("fooey"), "capture failed", capture))
		assert.Len(t, cov.Failures, 1)
		assert.Equal(t, `capture failed`, cov.Failures[0].Error.Error())
		assert.Contains(t, buf.String(), ": capture failed")
		assert.True(t, ctx.testing.Failed())
	})
}

func TestContext_reportUnmet(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
			return nil, nil
		})
		ctx.reportUnmet(exp, errors.New("fooey"))
		assert.Len(t, cov.Unmet, 1)
		assert.Equal(t, `fooey`, cov.Unmet[0].Error.Error())
		assert.Contains(t, buf.String(), ": fooey")
		assert.True(t, ctx.testing.Failed())
	})
	t.Run("basic, required", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		exp := &expectStatusCode{
			name:              "foo",
			expect:            200,
			frame:             framing.NewFrame(0),
			commonExpectation: commonExpectation{required: true},
		}
		ctx.reportUnmet(exp, errors.New("fooey"))
		assert.Len(t, cov.Unmet, 1)
		assert.Equal(t, `fooey`, cov.Unmet[0].Error.Error())
		assert.Contains(t, buf.String(), ": fooey")
		assert.True(t, ctx.testing.Failed())
	})
	t.Run("unmet error", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
			return nil, nil
		})
		umerr := &unmetError{
			msg:   "fooey",
			name:  "foo",
			frame: framing.NewFrame(0),
		}
		ctx.reportUnmet(exp, umerr)
		assert.Len(t, cov.Unmet, 1)
		assert.Equal(t, `fooey`, cov.Unmet[0].Error.Error())
		assert.Contains(t, buf.String(), ": fooey")
		assert.True(t, ctx.testing.Failed())
	})
	t.Run("unmet error, required", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		exp := &expectStatusCode{
			name:              "foo",
			expect:            200,
			frame:             framing.NewFrame(0),
			commonExpectation: commonExpectation{required: true},
		}
		umerr := &unmetError{
			msg:   "fooey",
			name:  "foo",
			frame: framing.NewFrame(0),
		}
		ctx.reportUnmet(exp, umerr)
		assert.Len(t, cov.Unmet, 1)
		assert.Equal(t, `fooey`, cov.Unmet[0].Error.Error())
		assert.Contains(t, buf.String(), ": fooey")
		assert.True(t, ctx.testing.Failed())
	})
}

func TestContext_reportMet(t *testing.T) {
	ctx := newContext()
	cov := coverage.NewCoverage()
	ctx.coverage = cov

	exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
		return nil, nil
	})
	ctx.reportMet(exp)
	assert.Len(t, cov.Met, 1)
}

func TestContext_reportSkipped(t *testing.T) {
	ctx := newContext()
	cov := coverage.NewCoverage()
	ctx.coverage = cov

	exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
		return nil, nil
	})
	ctx.reportSkipped(exp)
	assert.Len(t, cov.Skipped, 1)
}

func TestContext_currentTest(t *testing.T) {
	ctx := newContext()

	a := htesting.NewHelper(t, nil, nil)
	assert.Nil(t, ctx.currentTest())

	ctx.testing = a
	assert.NotNil(t, ctx.currentTest())
	assert.Equal(t, a, ctx.currentTest())

	b := htesting.NewHelper(t, nil, nil)
	ctx.currTesting = []htesting.Helper{a, b}
	assert.NotNil(t, ctx.currentTest())
	assert.Equal(t, b, ctx.currentTest())
}

func TestContext_run(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ok := ctx.run("foobar", &mockRunnable{})
		require.True(t, ok)
		assert.Len(t, cov.Failures, 0)
	})
	t.Run("basic, with error", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ok := ctx.run("foobar", &mockRunnable{err: errors.New("foo")})
		require.False(t, ok)
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("basic, with reported failure", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov

		ok := ctx.run("foobar", &mockRunnable{reportErr: errors.New("foo")})
		require.False(t, ok)
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("testing", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		ok := ctx.run("foobar", &mockRunnable{})
		require.True(t, ok)
		assert.Len(t, cov.Failures, 0)
		assert.False(t, ctx.testing.Failed())
	})
	t.Run("testing, with error", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		ok := ctx.run("foobar", &mockRunnable{err: errors.New("foo")})
		require.False(t, ok)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.testing.Failed())
	})
	t.Run("testing, with reported failure", func(t *testing.T) {
		ctx := newContext()
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)

		ok := ctx.run("foobar", &mockRunnable{reportErr: errors.New("foo")})
		require.False(t, ok)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.testing.Failed())
	})
}

func newTestContext(vars map[Var]any) *context {
	result := &context{
		coverage:     coverage.NewNullCoverage(),
		dbs:          make(namedDatabases),
		images:       make(map[string]with.Image),
		vars:         make(map[Var]any),
		cookieJar:    make(map[string]*http.Cookie),
		mockServices: make(map[string]service.MockedService),
	}
	for k, v := range vars {
		result.vars[k] = v
	}
	return result
}
