package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
)

func TestResolveValue(t *testing.T) {
	testCases := []struct {
		value          any
		ctx            *context
		dbMock         func(t *testing.T) *sql.DB
		body           any
		response       *http.Response
		responseCookie *http.Cookie
		expect         any
		expectErr      string
		setup          func(t *testing.T) func()
		setupCtx       func(ctx *context)
	}{
		{
			value: BodyReader(func(body any) (any, error) {
				return "foo", nil
			}),
			expect: "foo",
		},
		{
			value: func(body any) (any, error) {
				return "foo", nil
			},
			expect: "foo",
		},
		{
			value:     StatusCode(0),
			expectErr: "response is nil",
		},
		{
			value: StatusCode(0),
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			expect: http.StatusOK,
		},
		{
			value:     ResponseHeader("Content-Type"),
			expectErr: "response is nil",
		},
		{
			value: ResponseHeader("Content-Type"),
			response: &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
			},
			expect: "application/json",
		},
		{
			value: TemplateString("{$foo}..."),
			ctx: newTestContext(map[Var]any{
				"foo": 1,
			}),
			expect: "1...",
		},
		{
			value: TemplateString("{$foo}-{$bar}"),
			ctx: newTestContext(map[Var]any{
				"foo": 1,
				"bar": 2,
			}),
			expect: "1-2",
		},
		{
			value: TemplateString("\\{$foo}-{$bar}"),
			ctx: newTestContext(map[Var]any{
				"bar": 2,
			}),
			expect: "{$foo}-2",
		},
		{
			value: TemplateString("\\\\\\{$foo}-{$bar}"),
			ctx: newTestContext(map[Var]any{
				"foo": 1,
				"bar": 2,
			}),
			expect: "\\\\{$foo}-2",
		},
		{
			value: TemplateString("{$foo}-{$bar"),
			ctx: newTestContext(map[Var]any{
				"foo": 1,
			}),
			expect: "1-{$bar",
		},
		{
			value: TemplateString("{$foo}"),
			ctx: newTestContext(map[Var]any{
				"foo": Var("bar"),
			}),
			expectErr: "unknown variable \"bar\"",
		},
		{
			value: Var("foo"),
			ctx: newTestContext(map[Var]any{
				"foo": 42,
			}),
			expect: 42,
		},
		{
			value: Var("foo"),
			ctx: newTestContext(map[Var]any{
				"foo": Var("bar"),
				"bar": 42,
			}),
			expect: 42,
		},
		{
			value: DefaultVar("foo", Var("bar")),
			ctx: newTestContext(map[Var]any{
				"bar": 42,
			}),
			expect: 42,
		},
		{
			value: DefaultVar(Var("foo"), Var("bar")),
			ctx: newTestContext(map[Var]any{
				"bar": 42,
			}),
			expect: 42,
		},
		{
			value: DefaultVar(0, false),
			ctx: newTestContext(map[Var]any{
				"0": true,
			}),
			expect: true,
		},
		{
			value:     BodyPath("."),
			expectErr: "body is nil",
		},
		{
			value:  BodyPath("."),
			body:   map[string]any{"foo": "bar"},
			expect: map[string]any{"foo": "bar"},
		},
		{
			value:  JsonPath(map[string]any{"foo": "bar"}, "."),
			expect: map[string]any{"foo": "bar"},
		},
		{
			value:  JsonPath(map[string]any{"foo": "bar"}, "foo"),
			expect: "bar",
		},
		{
			value:     JsonPath(map[string]any{"foo": "bar"}, "xxx"),
			expectErr: "json path \"xxx\" does not exist",
		},
		{
			value:     JsonPath(nil, "foo"),
			expectErr: "json path \"foo\" into nil",
		},
		{
			value:     JsonPath("not an object or array", "foo"),
			expectErr: "json path \"foo\" into non object/array",
		},
		{
			value:  JsonPath([]any{}, "."),
			expect: []any{},
		},
		{
			value:     JsonPath([]any{}, FIRST),
			expectErr: "json path \"FIRST\" into empty array",
		},
		{
			value:     JsonPath([]any{}, LAST),
			expectErr: "json path \"LAST\" into empty array",
		},
		{
			value:     JsonPath([]any{}, "1"),
			expectErr: "json path \"1\" into empty array",
		},
		{
			value:     JsonPath([]any{nil}, "1"),
			expectErr: "json path \"1\" array index out of range",
		},
		{
			value:     JsonPath([]any{}, "not a number"),
			expectErr: "json path \"not a number\" invalid array index",
		},
		{
			value:     JsonPath(Var("test_var"), "."),
			ctx:       newTestContext(nil),
			expectErr: "unknown variable",
		},
		{
			value:  JsonPath(Var("test_var"), "foo"),
			ctx:    newTestContext(map[Var]any{"test_var": map[string]any{"foo": "bar"}}),
			expect: "bar",
		},
		{
			value:     Query("", "not a select"),
			expectErr: "db is nil",
		},
		{
			value: Query("", "not a select"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "query must start with \"SELECT\"",
		},
		{
			value: Query("", "SELECT * FROM table", Var("test_var")),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unknown variable \"test_var\"",
		},
		{
			value: Query("", "SELECT * FROM table WHERE foo = ?", Var("test_var")),
			ctx:   newTestContext(map[Var]any{"test_var": "bar"}),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WithArgs("bar").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: Query("", "SELECT * FROM {$unknown}"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unresolved variables in string",
		},
		{
			value: Query("", "SELECT * FROM table"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: Query("", "SELECT * FROM table"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo"}).AddRow("foo1"))
				return db
			},
			expect: "foo1",
		},
		{
			value:     QueryRows("", "not a select"),
			expectErr: "db is nil",
		},
		{
			value: QueryRows("", "not a select"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "query must start with \"SELECT\"",
		},
		{
			value: QueryRows("", "SELECT * FROM table", Var("test_var")),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unknown variable \"test_var\"",
		},
		{
			value: QueryRows("", "SELECT * FROM table WHERE foo = ?", Var("test_var")),
			ctx:   newTestContext(map[Var]any{"test_var": "bar"}),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WithArgs("bar").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: []map[string]any{
				{"foo": "foo1", "bar": int64(1)},
				{"foo": "foo2", "bar": int64(2)},
			},
		},
		{
			value: QueryRows("", "SELECT * FROM {$unknown}"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unresolved variables in string",
		},
		{
			value: QueryRows("", "SELECT * FROM table"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: []map[string]any{
				{"foo": "foo1", "bar": int64(1)},
				{"foo": "foo2", "bar": int64(2)},
			},
		},
		{
			value: JsonPath(QueryRows("", "SELECT * FROM table"), LAST),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo2", "bar": int64(2)},
		},
		{
			value: JsonPath(QueryRows("", "SELECT * FROM table"), FIRST),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: JsonPath(QueryRows("", "SELECT * FROM table"), LEN),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: 2,
		},
		{
			value: JsonPath(QueryRows("", "SELECT * FROM table"), "1"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2).AddRow("foo3", 3))
				return db
			},
			expect: map[string]any{"foo": "foo2", "bar": int64(2)},
		},
		{
			value: JsonPath(QueryRows("", "SELECT * FROM table"), "-1"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2).AddRow("foo3", 3))
				return db
			},
			expect: map[string]any{"foo": "foo3", "bar": int64(3)},
		},
		{
			value: JsonPath(JsonPath(QueryRows("", "SELECT * FROM table"), "-1"), "bar"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2).AddRow("foo3", 3))
				return db
			},
			expect: int64(3),
		},
		{
			value: JsonPath(JsonPath(JsonPath(Var("test_var"), LAST), "foo"), "-2"),
			ctx: newTestContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expect: 5,
		},
		{
			value: JsonTraverse(Var("test_var"), LAST, "foo", -2),
			ctx: newTestContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expect: 5,
		},
		{
			value: JsonTraverse(Var("test_var")),
			ctx: newTestContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expect: []any{
				map[string]any{"foo": []any{1, 2, 3}},
				map[string]any{"foo": []any{4, 5, 6}},
			},
		},
		{
			value: JsonTraverse(Var("test_var"), ".", LAST, ".", "foo", "", -2, ".", true),
			ctx: newTestContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expectErr: "failed to traverse json path \"LAST.foo[-2].true\": json path \"true\" into non object/array",
		},
		{
			value:    JsonPath(ResponseCookie("session"), "Value"),
			response: &http.Response{},
			responseCookie: &http.Cookie{
				Name:  "session",
				Value: "test session",
			},
			expect: "test session",
		},
		{
			value:     JsonPath(ResponseCookie("session"), "Value"),
			expectErr: "response is nil",
		},
		{
			value: map[string]any{
				"foo": Var("foo"),
			},
			ctx: newTestContext(map[Var]any{
				"foo": Var("bar"),
				"bar": 42,
			}),
			expect: map[string]any{"foo": 42},
		},
		{
			value: map[string]any{
				"foo": Var("foo"),
			},
			expectErr: "unknown variable \"foo\"",
		},
		{
			value: []any{Var("foo")},
			ctx: newTestContext(map[Var]any{
				"foo": Var("bar"),
				"bar": 42,
			}),
			expect: []any{42},
		},
		{
			value:     []any{Var("foo")},
			expectErr: "unknown variable \"foo\"",
		},
		{
			value: map[string]any{
				"foo": []any{Var("foo")},
			},
			ctx: newTestContext(map[Var]any{
				"foo": Var("bar"),
				"bar": 42,
			}),
			expect: map[string]any{"foo": []any{42}},
		},
		{
			value:  JSON{"foo": 42},
			expect: map[string]any{"foo": 42},
		},
		{
			value:  JSONArray{"foo", 42},
			expect: []any{"foo", 42},
		},
		{
			value:  Body,
			expect: nil,
		},
		{
			value:  Body,
			body:   map[string]any{"foo": 42},
			expect: map[string]any{"foo": 42},
		},
		{
			value:  Env("TEST_ENV"),
			expect: "",
			setup: func(t *testing.T) func() {
				_ = os.Unsetenv("TEST_ENV")
				return func() {}
			},
		},
		{
			value: Env("TEST_ENV"),
			setup: func(t *testing.T) func() {
				_ = os.Setenv("TEST_ENV", "test")
				return func() {
					_ = os.Unsetenv("TEST_ENV")
				}
			},
			expect: "test",
		},
		{
			value:     TemplateString("{$env:TEST_ENV}"),
			expectErr: "unresolved env var: ",
		},
		{
			value: TemplateString("{$env:TEST_ENV}"),
			setup: func(t *testing.T) func() {
				_ = os.Setenv("TEST_ENV", "test")
				return func() {
					_ = os.Unsetenv("TEST_ENV")
				}
			},
			expect: "test",
		},
		{
			value:  Len("foo"),
			expect: 3,
		},
		{
			value:  Len(map[string]any{"foo": 42}),
			expect: 1,
		},
		{
			value:  Len([]any{"foo", 42}),
			expect: 2,
		},
		{
			value:  Len([]string{"foo", "bar"}),
			expect: 2,
		},
		{
			value:  Len(true),
			expect: -1,
		},
		{
			value:     ApiLogs(10),
			expectErr: "no api image for logs",
		},
		{
			value: ApiLogs(10),
			setupCtx: func(ctx *context) {
				ctx.apiImage = &mockApiImage{}
			},
			expectErr: "no api image for logs",
		},
		{
			value: func() (any, error) {
				return "foo", nil
			},
			expect: "foo",
		},
		{
			value: func() (any, error) {
				return nil, errors.New("fooey")
			},
			expectErr: "fooey",
		},
		{
			value:     TemplateString("{$svc:foo:host}"),
			expectErr: "unresolved service var:",
		},
		{
			value: TemplateString("{$svc:foo:host}"),
			setupCtx: func(ctx *context) {
				ctx.images["foo"] = &mockImage{}
			},
			expect: "localhost",
		},
		{
			value: TemplateString("{$svc:foo:host}--{$svc:foo:port}--{$svc:foo:mport}--{$svc:foo:username}--{$svc:foo:password}--{$svc:foo:region}"),
			setupCtx: func(ctx *context) {
				ctx.images["foo"] = &mockImage{
					envs: map[string]string{
						"region": "us-east-1",
					},
				}
			},
			expect: "localhost--8080--50080--foo--bar--us-east-1",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = newTestContext(nil)
			}
			if tc.setupCtx != nil {
				tc.setupCtx(ctx)
			}
			ctx.currResponse = tc.response
			ctx.currBody = tc.body
			if tc.dbMock != nil {
				db := tc.dbMock(t)
				ctx.dbs.register("", db, common.DatabaseArgs{})
				defer db.Close()
			}
			if tc.setup != nil {
				td := tc.setup(t)
				if td != nil {
					defer td()
				}
			}
			if tc.response != nil && tc.responseCookie != nil {
				if tc.response.Header == nil {
					tc.response.Header = http.Header{}
				}
				tc.response.Header.Add("Set-Cookie", tc.responseCookie.String())
			}
			av, err := ResolveValue(tc.value, ctx)
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expect, av)
			}
		})
	}
}

func TestResolveValues(t *testing.T) {
	ctx := newTestContext(map[Var]any{"foo": "bar", "bar": 42})
	av1, av2, err := ResolveValues(Var("foo"), Var("bar"), ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", av1)
	assert.Equal(t, 42, av2)
}

func TestResolveData(t *testing.T) {
	testCases := []struct {
		value  any
		expect string
	}{
		{
			expect: "",
		},
		{
			value:  []byte("foo"),
			expect: "foo",
		},
		{
			value:  "foo",
			expect: "foo",
		},
		{
			value:  JSON{"foo": 42},
			expect: `{"foo":42}`,
		},
		{
			value:  42,
			expect: "42",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			ctx := newTestContext(nil)
			data, err := ResolveData(tc.value, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.expect, string(data))
		})
	}
}

func TestJsonify(t *testing.T) {
	testCases := []struct {
		ctx       Context
		value     any
		expect    any
		expectErr string
	}{
		{
			value:  Jsonify(nil),
			expect: nil,
		},
		{
			value:  Jsonify(`{"foo": "bar"}`),
			expect: map[string]any{"foo": "bar"},
		},
		{
			value:     Jsonify(`{"foo": `),
			expectErr: "unexpected end of JSON input",
		},
		{
			value:  Jsonify(`["foo", "bar"]`),
			expect: []any{"foo", "bar"},
		},
		{
			value:  Jsonify([]byte(`{"foo": "bar"}`)),
			expect: map[string]any{"foo": "bar"},
		},
		{
			value:     Jsonify([]byte(`{"foo": `)),
			expectErr: "unexpected end of JSON input",
		},
		{
			value:  Jsonify([]byte(`["foo", "bar"]`)),
			expect: []any{"foo", "bar"},
		},
		{
			value:  Jsonify(map[string]any{"foo": "bar"}),
			expect: map[string]any{"foo": "bar"},
		},
		{
			value:  Jsonify([]any{"foo", "bar"}),
			expect: []any{"foo", "bar"},
		},
		{
			value:  Jsonify([]string{"foo", "bar"}),
			expect: []any{"foo", "bar"},
		},
		{
			value:     Jsonify(true),
			expectErr: "invalid type for json coerce: ",
		},
		{
			value:  First([]any{1, 2, 3}),
			expect: 1,
		},
		{
			value:  First([]any{}),
			expect: nil,
		},
		{
			value:  First([]string{"foo", "bar"}),
			expect: "foo",
		},
		{
			value:  First([]string{}),
			expect: nil,
		},
		{
			value:  Last([]any{1, 2, 3}),
			expect: 3,
		},
		{
			value:  Last([]any{}),
			expect: nil,
		},
		{
			value:  Last([]string{"foo", "bar"}),
			expect: "bar",
		},
		{
			value:  Last([]string{}),
			expect: nil,
		},
		{
			value:  Nth([]any{1, 2, 3}, 1),
			expect: 2,
		},
		{
			value:  Nth([]any{1, 2, 3}, -2),
			expect: 2,
		},
		{
			value:  Nth([]string{"foo", "bar", "baz"}, 1),
			expect: "bar",
		},
		{
			value:  Nth([]string{"foo", "bar", "baz"}, -2),
			expect: "bar",
		},
		{
			value:  And(true),
			expect: true,
		},
		{
			value:  And(true, false),
			expect: false,
		},
		{
			value:     And("not a bool"),
			expectErr: "and value expects boolean - got type ",
		},
		{
			value:  And(nil, Var("foo")),
			ctx:    newTestContext(map[Var]any{"foo": true}),
			expect: true,
		},
		{
			value:  And(Var("foo"), ExpectEqual(0, 1)),
			ctx:    newTestContext(map[Var]any{"foo": true}),
			expect: false,
		},
		{
			value:  And(Var("foo"), ExpectEqual(0, 0)),
			ctx:    newTestContext(map[Var]any{"foo": true}),
			expect: true,
		},
		{
			value:     And(nil, Var("foo")),
			expectErr: "unknown variable ",
		},
		{
			value:     And(true, ExpectEqual(0, Var("foo"))),
			expectErr: "unknown variable ",
		},
		{
			value:  Or(false),
			expect: false,
		},
		{
			value:  Or(false, true),
			expect: true,
		},
		{
			value:     Or("not a bool"),
			expectErr: "or value expects boolean - got type ",
		},
		{
			value:  Or(nil, Var("foo")),
			ctx:    newTestContext(map[Var]any{"foo": true}),
			expect: true,
		},
		{
			value:  Or(Var("foo"), ExpectEqual(0, 0)),
			ctx:    newTestContext(map[Var]any{"foo": false}),
			expect: true,
		},
		{
			value:  Or(Var("foo"), ExpectEqual(1, 0)),
			ctx:    newTestContext(map[Var]any{"foo": false}),
			expect: false,
		},
		{
			value:     Or(nil, Var("foo")),
			expectErr: "unknown variable ",
		},
		{
			value:     Or(false, ExpectEqual(0, Var("foo"))),
			expectErr: "unknown variable ",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = newTestContext(map[Var]any{})
			}
			av, err := ResolveValue(tc.value, ctx)
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expect, av)
			}
		})
	}
}

func Test_stringifyValue(t *testing.T) {
	testCases := []struct {
		value  any
		expect string
	}{
		{
			value:  nil,
			expect: "<nil>",
		},
		{
			value:  "foo",
			expect: `"foo"`,
		},
		{
			value:  42,
			expect: "42",
		},
		{
			value:  Var("foo"),
			expect: `Var(foo)`,
		},
		{
			value:  DefaultVar("foo", Var("bar")),
			expect: `DefaultVar(foo, Var(bar))`,
		},
		{
			value:  Body,
			expect: `Body`,
		},
		{
			value:  Env("TEST_ENV"),
			expect: `Env(TEST_ENV)`,
		},
		{
			value:  QueryValue{Query: "SELECT *", Args: []any{"foo"}},
			expect: `Query("SELECT *", "foo")`,
		},
		{
			value:  JsonPathValue{Value: Var("test"), Path: "."},
			expect: `JsonPath(Var(test), ".")`,
		},
		{
			value:  ApiLogs(10),
			expect: `ApiLogs(10)`,
		},
		{
			value:  Len(Var("foo")),
			expect: `Len(Var(foo))`,
		},
		{
			value:  First(Var("foo")),
			expect: `First(Var(foo))`,
		},
		{
			value:  Last(Var("foo")),
			expect: `Last(Var(foo))`,
		},
		{
			value:  Nth(Var("foo"), -1),
			expect: `Nth(Var(foo), -1)`,
		},
		{
			value:  And(Var("foo"), ExpectEqual(0, 0)),
			expect: `And(Var(foo), ExpectEqual)`,
		},
		{
			value:  Or(Var("foo"), ExpectEqual(0, 0)),
			expect: `Or(Var(foo), ExpectEqual)`,
		},
		{
			value:  ApiCall(GET, "/api/foo", nil, nil),
			expect: `ApiCall(GET /api/foo)`,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			assert.Equal(t, tc.expect, stringifyValue(tc.value))
		})
	}
}

func TestApiCall(t *testing.T) {
	ctx := newTestContext(map[Var]any{
		"foo": "fooey",
		"id":  123,
		"ct":  "application/json",
	})
	ctx.host = "localhost:8080"
	ctx.httpDo = &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo": "bar"}`),
		hdrs: map[string]string{
			"Content-Type": "application/json",
		},
	}
	t.Run("json body", func(t *testing.T) {
		rv := ApiCall(GET, TemplateString("/api/foo/{$id}"), JSON{"foo": Var("foo")}, Headers{"Content-Type": Var("ct")})

		v, err := ResolveValue(rv, ctx)
		require.NoError(t, err)
		mv := v.(map[string]any)
		assert.Equal(t, http.StatusOK, mv["status"])
		assert.Equal(t, []byte(`{"foo": "bar"}`), mv["body"].([]byte))
		assert.Equal(t, "application/json", mv["headers"].(map[string]any)["Content-Type"])
	})
	t.Run("[]byte body", func(t *testing.T) {
		rv := ApiCall(GET, TemplateString("/api/foo/{$id}"), []byte(`{"foo": "bar"}`), Headers{"Content-Type": Var("ct")})

		_, err := ResolveValue(rv, ctx)
		require.NoError(t, err)
	})
	t.Run("string body", func(t *testing.T) {
		rv := ApiCall(GET, TemplateString("/api/foo/{$id}"), `{"foo": "bar"}`, Headers{"Content-Type": Var("ct")})

		_, err := ResolveValue(rv, ctx)
		require.NoError(t, err)
	})
	t.Run("int body", func(t *testing.T) {
		rv := ApiCall(GET, TemplateString("/api/foo/{$id}"), Var("id"), Headers{"Content-Type": Var("ct")})

		_, err := ResolveValue(rv, ctx)
		require.NoError(t, err)
	})
}
