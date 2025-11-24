package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
)

func TestSetVar(t *testing.T) {
	t.Run("normal string name", func(t *testing.T) {
		c := SetVar("foo", "bar")
		assert.Equal(t, "foo", c.Name())
		assert.NotNil(t, c.Frame())

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("var name", func(t *testing.T) {
		c := SetVar(Var("foo"), "bar")
		assert.Equal(t, "foo", c.Name())
		assert.NotNil(t, c.Frame())

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("any name", func(t *testing.T) {
		c := SetVar(0, "bar")
		assert.Equal(t, "0", c.Name())
		assert.NotNil(t, c.Frame())

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["0"])
	})
	t.Run("func", func(t *testing.T) {
		c := SetVar("foo", "bar")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
}

func TestClearVars(t *testing.T) {
	c := ClearVars()
	assert.Equal(t, "CLEAR VARS", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(map[Var]any{"foo": "bar"})
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Empty(t, ctx.vars)
}

func TestDbInsert(t *testing.T) {
	c := DbInsert("", "table", Columns{"foo": "bar"})
	assert.Equal(t, "INSERT table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
	err = c.Run(ctx)
	require.NoError(t, err)
}

func TestDbExec(t *testing.T) {
	c := DbExec("", "DELETE FROM table")
	assert.Equal(t, "EXEC DELETE FROM table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
	err = c.Run(ctx)
	require.NoError(t, err)
}

func TestDbClearTable(t *testing.T) {
	c := DbClearTable("", "table")
	assert.Equal(t, "DELETE FROM table", c.Name())
	assert.NotNil(t, c.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
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

func TestSetCookie(t *testing.T) {
	c := SetCookie(&http.Cookie{Name: "foo", Value: "bar"})
	assert.Equal(t, "SET COOKIE foo", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.cookieJar, 1)

	c2 := SetCookie(&http.Cookie{Name: "foo2", Value: "bar"})
	err = c2.Run(ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.cookieJar, 2)
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

func TestMockServicesClearAll(t *testing.T) {
	c := MockServicesClearAll()
	assert.Equal(t, "CLEAR ALL MOCK SERVICES", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.cleared)
}

func TestMockServiceClear(t *testing.T) {
	c := MockServiceClear("mock")
	assert.Equal(t, "CLEAR MOCK SERVICE [mock]", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.cleared)

	c = &mockServiceClear{
		name:  "unknown",
		frame: framing.NewFrame(0),
	}
	err = c.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mock service ")
}

func TestMockServiceCall(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c := MockServiceCall(
			"mock",
			"/foos/{$id}",
			http.MethodGet,
			200,
			map[string]any{
				"foo": Var("foo"),
			},
			"X-Hdr-Bar", Var("bar"))

		ctx := newTestContext(map[Var]any{
			"id":  "123",
			"foo": "foo-value",
			"bar": "bar-value",
		})
		ms := &mockMockedService{}
		ctx.mockServices["mock"] = ms
		err := c.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("missing var in path", func(t *testing.T) {
		c := MockServiceCall(
			"mock",
			"/foos/{$id}",
			http.MethodGet,
			200,
			map[string]any{
				"foo": Var("foo"),
			},
			[]any{"X-Hdr-Bar", Var("bar")})

		ctx := newTestContext(map[Var]any{
			"foo": "foo-value",
			"bar": "bar-value",
		})
		ms := &mockMockedService{}
		ctx.mockServices["mock"] = ms
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved variables in string ")
	})
	t.Run("missing var in body", func(t *testing.T) {
		c := MockServiceCall(
			"mock",
			"/foos/{$id}",
			http.MethodGet,
			200,
			map[string]any{
				"foo": Var("foo"),
			},
			[]any{"X-Hdr-Bar", Var("bar")})

		ctx := newTestContext(map[Var]any{
			"id":  "123",
			"bar": "bar-value",
		})
		ms := &mockMockedService{}
		ctx.mockServices["mock"] = ms
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown variable")
	})
	t.Run("missing var in headers", func(t *testing.T) {
		c := MockServiceCall(
			"mock",
			"/foos/{$id}",
			http.MethodGet,
			200,
			map[string]any{
				"foo": Var("foo"),
			},
			"X-Hdr-Bar", Var("bar"))

		ctx := newTestContext(map[Var]any{
			"id":  "123",
			"foo": "foo-value",
		})
		ms := &mockMockedService{}
		ctx.mockServices["mock"] = ms
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown variable")
	})
	t.Run("unknown mock", func(t *testing.T) {
		c := MockServiceCall(
			"mock",
			"/foos",
			http.MethodGet,
			200, nil)

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown mock service ")
	})
}

func TestWait(t *testing.T) {
	c := Wait(10)
	assert.Equal(t, "WAIT 10ms", c.Name())
	assert.NotNil(t, c.Frame())
	require.NoError(t, c.Run(nil))
}

func TestSetEnv(t *testing.T) {
	c := SetEnv("TEST_ENV", "foo")
	assert.Equal(t, "SET ENV: \"TEST_ENV\"", c.Name())
	assert.NotNil(t, c.Frame())
	_ = os.Unsetenv("TEST_ENV")
	err := c.Run(nil)
	require.NoError(t, err)
	s, ok := os.LookupEnv("TEST_ENV")
	assert.True(t, ok)
	assert.Equal(t, "foo", s)

	c = SetEnv("TEST_ENV", 42)
	err = c.Run(nil)
	require.NoError(t, err)
	s, ok = os.LookupEnv("TEST_ENV")
	assert.True(t, ok)
	assert.Equal(t, "42", s)
}

func TestUnSetEnv(t *testing.T) {
	c := UnSetEnv("TEST_ENV", "TEST_ENV2")
	assert.Equal(t, "UNSET ENV: \"TEST_ENV\", \"TEST_ENV2\"", c.Name())
	assert.NotNil(t, c.Frame())

	_ = os.Setenv("TEST_ENV", "foo")
	_ = os.Unsetenv("TEST_ENV2")
	err := c.Run(nil)
	require.NoError(t, err)
	_, ok := os.LookupEnv("TEST_ENV")
	assert.False(t, ok)
	_, ok = os.LookupEnv("TEST_ENV2")
	assert.False(t, ok)
}

func Test_conditional(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := &conditional{
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "CONDITIONAL", c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("bool condition", func(t *testing.T) {
		c := &conditional{
			condition: true,
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("bool var condition", func(t *testing.T) {
		c := &conditional{
			condition: Var("foo"),
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(map[Var]any{"foo": true})
		err := c.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("var condition - missing var", func(t *testing.T) {
		c := &conditional{
			condition: Var("foo"),
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown variable")
	})
	t.Run("invalid condition", func(t *testing.T) {
		c := &conditional{
			condition: "not a bool",
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid condition type: ")
	})
	t.Run("expectation condition", func(t *testing.T) {
		c := &conditional{
			condition: ExpectEqual(0, 0),
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("successful capture ops", func(t *testing.T) {
		c := &conditional{
			condition: true,
			ops: []Runnable{
				nil,
				SetVar("foo", "bar"),
			},
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("successful expectation ops", func(t *testing.T) {
		c := &conditional{
			condition: true,
			ops: []Runnable{
				ExpectEqual(0, 0),
			},
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("expectation condition unmet - ops not run, no failure", func(t *testing.T) {
		c := &conditional{
			condition: ExpectEqual(1, 0),
			ops: []Runnable{
				SetVar("foo", "bar"),
			},
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ctx.coverage = coverage.NewCoverage()

		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Nil(t, ctx.vars["foo"])
		assert.False(t, ctx.coverage.HasFailures())
	})
	t.Run("unsuccessful expectation unmet", func(t *testing.T) {
		c := &conditional{
			condition: true,
			ops: []Runnable{
				ExpectEqual(1, 0),
				ExpectEqual(0, 0),
			},
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ctx.coverage = coverage.NewCoverage()
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.True(t, ctx.coverage.HasFailures())
		cov := ctx.coverage.(*coverage.Coverage)
		assert.Len(t, cov.Common.Met, 0)
		assert.Len(t, cov.Common.Unmet, 1)
		assert.Len(t, cov.Common.Skipped, 1)
	})
	t.Run("unsuccessful expectation failure", func(t *testing.T) {
		c := &conditional{
			condition: true,
			ops: []Runnable{
				ExpectEqual(1, Var("foo")),
				ExpectEqual(0, 0),
			},
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ctx.coverage = coverage.NewCoverage()
		err := c.Run(ctx)
		require.Error(t, err)
		assert.True(t, ctx.coverage.HasFailures())
		cov := ctx.coverage.(*coverage.Coverage)
		assert.Len(t, cov.Common.Met, 0)
		assert.Len(t, cov.Common.Unmet, 0)
		assert.Len(t, cov.Common.Failures, 1)
		assert.Len(t, cov.Common.Skipped, 1)
	})
	t.Run("not bool successful", func(t *testing.T) {
		c := &conditional{
			condition: false,
			not:       true,
			ops:       []Runnable{SetVar("foo", "bar")},
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("not expectation successful", func(t *testing.T) {
		c := &conditional{
			condition: ExpectVarSet(Var("foo")),
			not:       true,
			ops:       []Runnable{SetVar("foo", "bar")},
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("If func", func(t *testing.T) {
		c := If(true, SetVar("foo", "bar"))
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("IfNot func", func(t *testing.T) {
		c := IfNot(false, SetVar("foo", "bar"))
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("If func - nested", func(t *testing.T) {
		c := If(true, If(true, If(true, SetVar("foo", "bar"))))
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
}

func Test_forEach(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := &forEach{
			value: JsonPath(Var("foo"), "."),
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, `ForEach(JsonPath(Var(foo), "."))`, c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("errors with invalid value", func(t *testing.T) {
		c := ForEach("", "")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
	})
	t.Run("successful expect ops", func(t *testing.T) {
		c := ForEach(Var("foo"), ".",
			nil,
			ExpectEqual(Var("."), 0),
			ExpectNotEqual(Var("."), 1),
			SetVar("last", Var(".")))
		ctx := newTestContext(map[Var]any{"foo": []any{0, 0, 0}})
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, ctx.vars["last"])
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Met, 6)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Failures, 0)
	})
	t.Run("ops unmet", func(t *testing.T) {
		c := ForEach(Var("foo"), ".",
			ExpectEqual(Var("."), 0))
		ctx := newTestContext(map[Var]any{"foo": []any{0, 1, 2}})
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Met, 1)
		assert.Len(t, cov.Unmet, 2)
		assert.Len(t, cov.Failures, 0)
		assert.Len(t, cov.Skipped, 0)
	})
	t.Run("ops failure", func(t *testing.T) {
		c := ForEach(Var("foo"), ".",
			ExpectEqual(Var("unknown"), 0),
			ExpectEqual(Var("."), 0))
		ctx := newTestContext(map[Var]any{"foo": []any{0, 1, 2}})
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		err := c.Run(ctx)
		require.Error(t, err)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Met, 0)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Failures, 3)
		assert.Len(t, cov.Skipped, 3)
	})
	t.Run("other slice type", func(t *testing.T) {
		c := ForEach(Var("foo"), ".", SetVar("last", Var(".")))
		ctx := newTestContext(map[Var]any{"foo": []string{"foo", "bar", "baz"}})
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "baz", ctx.vars["last"])
	})
	t.Run("other iter var type", func(t *testing.T) {
		c := ForEach(Var("foo"), 42, SetVar("last", Var("42")))
		ctx := newTestContext(map[Var]any{"foo": []string{"foo", "bar", "baz"}})
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "baz", ctx.vars["last"])
	})
	t.Run("other iter var type - var", func(t *testing.T) {
		c := ForEach(Var("foo"), Var("."), SetVar("last", Var(".")))
		ctx := newTestContext(map[Var]any{"foo": []string{"foo", "bar", "baz"}})
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "baz", ctx.vars["last"])
	})
	t.Run("other iter var type - nil", func(t *testing.T) {
		c := ForEach(Var("foo"), nil, SetVar("last", Var(".")))
		ctx := newTestContext(map[Var]any{"foo": []string{"foo", "bar", "baz"}})
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "baz", ctx.vars["last"])
	})
}
