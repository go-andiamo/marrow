package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func Test_setVar(t *testing.T) {
	t.Run("normal string name", func(t *testing.T) {
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
	})
	t.Run("var name", func(t *testing.T) {
		c := &setVar{
			name:  Var("foo"),
			value: "bar",
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "foo", c.Name())
		assert.NotNil(t, c.Frame())

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
	t.Run("any name", func(t *testing.T) {
		c := &setVar{
			name:  0,
			value: "bar",
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "0", c.Name())
		assert.NotNil(t, c.Frame())

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["0"])
	})
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
	ctx.dbs.register("", db, common.DatabaseArgs{})
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
	ctx.dbs.register("", db, common.DatabaseArgs{})
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

func Test_mockServicesClearAll(t *testing.T) {
	c := &mockServicesClearAll{
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "CLEAR ALL MOCK SERVICES", c.Name())
	assert.NotNil(t, c.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := c.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.cleared)
}

func Test_mockServiceClear(t *testing.T) {
	c := &mockServiceClear{
		name:  "mock",
		frame: framing.NewFrame(0),
	}
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

func Test_mockServiceCall(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c := &mockServiceCall{
			name:           "mock",
			path:           "/foos/{$id}",
			method:         http.MethodGet,
			responseStatus: 200,
			responseBody: map[string]any{
				"foo": Var("foo"),
			},
			headers: []any{"X-Hdr-Bar", Var("bar")},
			frame:   framing.NewFrame(0),
		}

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
		c := &mockServiceCall{
			name:           "mock",
			path:           "/foos/{$id}",
			method:         http.MethodGet,
			responseStatus: 200,
			responseBody: map[string]any{
				"foo": Var("foo"),
			},
			headers: []any{"X-Hdr-Bar", Var("bar")},
			frame:   framing.NewFrame(0),
		}

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
		c := &mockServiceCall{
			name:           "mock",
			path:           "/foos/{$id}",
			method:         http.MethodGet,
			responseStatus: 200,
			responseBody: map[string]any{
				"foo": Var("foo"),
			},
			headers: []any{"X-Hdr-Bar", Var("bar")},
			frame:   framing.NewFrame(0),
		}

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
		c := &mockServiceCall{
			name:           "mock",
			path:           "/foos/{$id}",
			method:         http.MethodGet,
			responseStatus: 200,
			responseBody: map[string]any{
				"foo": Var("foo"),
			},
			headers: []any{"X-Hdr-Bar", Var("bar")},
			frame:   framing.NewFrame(0),
		}

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
		c := &mockServiceCall{
			name:           "mock",
			path:           "/foos",
			method:         http.MethodGet,
			responseStatus: 200,
			frame:          framing.NewFrame(0),
		}

		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown mock service ")
	})
}

func Test_wait(t *testing.T) {
	c := &wait{
		ms:    10,
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "WAIT 10ms", c.Name())
	assert.NotNil(t, c.Frame())
}

func Test_setEnv(t *testing.T) {
	c := &setEnv{
		name:  "TEST_ENV",
		value: "foo",
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "SET ENV: \"TEST_ENV\"", c.Name())
	assert.NotNil(t, c.Frame())
}

func Test_unSetEnv(t *testing.T) {
	c := &unSetEnv{
		names: []string{"TEST_ENV", "TEST_ENV2"},
		frame: framing.NewFrame(0),
	}
	assert.Equal(t, "UNSET ENV: \"TEST_ENV\", \"TEST_ENV2\"", c.Name())
	assert.NotNil(t, c.Frame())
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
				SetVar(Before, "foo", "bar"),
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
				SetVar(Before, "foo", "bar"),
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
		assert.Len(t, cov.Common.Failures, 1)
		assert.Len(t, cov.Common.Skipped, 1)
	})
	t.Run("not bool successful", func(t *testing.T) {
		c := &conditional{
			condition: false,
			not:       true,
			ops:       []Runnable{SetVar(Before, "foo", "bar")},
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
			ops:       []Runnable{SetVar(Before, "foo", "bar")},
			frame:     framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.vars["foo"])
	})
}
