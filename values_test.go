package marrow

import (
	"database/sql"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
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
			ctx: newContext(map[Var]any{
				"foo": 1,
			}),
			expect: "1...",
		},
		{
			value: TemplateString("{$foo}-{$bar}"),
			ctx: newContext(map[Var]any{
				"foo": 1,
				"bar": 2,
			}),
			expect: "1-2",
		},
		{
			value: TemplateString("\\{$foo}-{$bar}"),
			ctx: newContext(map[Var]any{
				"bar": 2,
			}),
			expect: "{$foo}-2",
		},
		{
			value: TemplateString("\\\\\\{$foo}-{$bar}"),
			ctx: newContext(map[Var]any{
				"foo": 1,
				"bar": 2,
			}),
			expect: "\\\\{$foo}-2",
		},
		{
			value: TemplateString("{$foo}-{$bar"),
			ctx: newContext(map[Var]any{
				"foo": 1,
			}),
			expect: "1-{$bar",
		},
		{
			value: TemplateString("{$foo}"),
			ctx: newContext(map[Var]any{
				"foo": Var("bar"),
			}),
			expectErr: "unknown variable \"bar\"",
		},
		{
			value: Var("foo"),
			ctx: newContext(map[Var]any{
				"foo": 42,
			}),
			expect: 42,
		},
		{
			value: Var("foo"),
			ctx: newContext(map[Var]any{
				"foo": Var("bar"),
				"bar": 42,
			}),
			expect: 42,
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
			ctx:       newContext(nil),
			expectErr: "unknown variable",
		},
		{
			value:  JsonPath(Var("test_var"), "foo"),
			ctx:    newContext(map[Var]any{"test_var": map[string]any{"foo": "bar"}}),
			expect: "bar",
		},
		{
			value:     Query("not a select"),
			expectErr: "db is nil",
		},
		{
			value: Query("not a select"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "query must start with \"SELECT\"",
		},
		{
			value: Query("SELECT * FROM table", Var("test_var")),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unknown variable \"test_var\"",
		},
		{
			value: Query("SELECT * FROM table WHERE foo = ?", Var("test_var")),
			ctx:   newContext(map[Var]any{"test_var": "bar"}),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WithArgs("bar").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: Query("SELECT * FROM {$unknown}"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unresolved variables in string",
		},
		{
			value: Query("SELECT * FROM table"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: Query("SELECT * FROM table"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo"}).AddRow("foo1"))
				return db
			},
			expect: "foo1",
		},
		{
			value:     QueryRows("not a select"),
			expectErr: "db is nil",
		},
		{
			value: QueryRows("not a select"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "query must start with \"SELECT\"",
		},
		{
			value: QueryRows("SELECT * FROM table", Var("test_var")),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unknown variable \"test_var\"",
		},
		{
			value: QueryRows("SELECT * FROM table WHERE foo = ?", Var("test_var")),
			ctx:   newContext(map[Var]any{"test_var": "bar"}),
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
			value: QueryRows("SELECT * FROM {$unknown}"),
			dbMock: func(t *testing.T) *sql.DB {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				return db
			},
			expectErr: "unresolved variables in string",
		},
		{
			value: QueryRows("SELECT * FROM table"),
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
			value: JsonPath(QueryRows("SELECT * FROM table"), LAST),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo2", "bar": int64(2)},
		},
		{
			value: JsonPath(QueryRows("SELECT * FROM table"), FIRST),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: map[string]any{"foo": "foo1", "bar": int64(1)},
		},
		{
			value: JsonPath(QueryRows("SELECT * FROM table"), LEN),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2))
				return db
			},
			expect: 2,
		},
		{
			value: JsonPath(QueryRows("SELECT * FROM table"), "1"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2).AddRow("foo3", 3))
				return db
			},
			expect: map[string]any{"foo": "foo2", "bar": int64(2)},
		},
		{
			value: JsonPath(QueryRows("SELECT * FROM table"), "-1"),
			dbMock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo", "bar"}).AddRow("foo1", 1).AddRow("foo2", 2).AddRow("foo3", 3))
				return db
			},
			expect: map[string]any{"foo": "foo3", "bar": int64(3)},
		},
		{
			value: JsonPath(JsonPath(QueryRows("SELECT * FROM table"), "-1"), "bar"),
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
			ctx: newContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expect: 5,
		},
		{
			value: JsonTraverse(Var("test_var"), LAST, "foo", -2),
			ctx: newContext(map[Var]any{
				"test_var": []any{
					map[string]any{"foo": []any{1, 2, 3}},
					map[string]any{"foo": []any{4, 5, 6}},
				},
			}),
			expect: 5,
		},
		{
			value: JsonTraverse(Var("test_var")),
			ctx: newContext(map[Var]any{
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
			ctx: newContext(map[Var]any{
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
			ctx: newContext(map[Var]any{
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
			ctx: newContext(map[Var]any{
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
			ctx: newContext(map[Var]any{
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
			value:     Body,
			expectErr: "body is nil",
		},
		{
			value:  Body,
			body:   map[string]any{"foo": 42},
			expect: map[string]any{"foo": 42},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = newContext(nil)
			}
			ctx.currResponse = tc.response
			ctx.currBody = tc.body
			if tc.dbMock != nil {
				ctx.db = tc.dbMock(t)
				defer ctx.db.Close()
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

func Test_stringifyValue(t *testing.T) {
	assert.Equal(t, "<nil>", stringifyValue(nil))
	assert.Equal(t, `"test"`, stringifyValue("test"))
	assert.Equal(t, `Var(test)`, stringifyValue(Var("test")))
	assert.Equal(t, "42", stringifyValue(42))
	assert.Equal(t, `Query("SELECT *", "foo")`, stringifyValue(QueryValue{Query: "SELECT *", Args: []any{"foo"}}))
	assert.Equal(t, `JsonPath(Var(test), ".")`, stringifyValue(JsonPathValue{Value: Var("test"), Path: "."}))
	assert.Equal(t, "Body", stringifyValue(Body))
}
