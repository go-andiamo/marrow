package marrow

import (
	"bytes"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestSuite(t *testing.T) {
	t.Skip()
	s := Suite(
		Endpoint("/foos", "Foos endpoint",
			SetVar(Before, "rows", QueryRows("SELECT * FROM foos")),
			SetVar(Before, "row", JsonPath(Var("rows"), LAST)),
			Method(GET, "Get foos").
				AssertOK().
				AssertEqual(3, JsonPath(Var("rows"), LEN)).
				AssertLen(Var("rows"), 3).
				AssertLen(Body, 1).
				AssertEqual(JsonPath(Var("row"), "foo_col"), "foo3").
				AssertStatus(Var("OK")).
				//SetVar(Before, "z", Query("xxx", Var("yyy"), Query("zzz"))).
				SetVar(After, "body", BodyPath(".")).
				SetVar(After, "foo", JsonPath(Var("body"), "foo")).
				AssertEqual(Var("foo"), "xxx").
				AssertEqual(JsonPath(Var("body"), "foo"), "xxx").
				AssertEqual(JsonPath(Var("body"), "foo"), 123.1).
				FailFast(),
			Method(POST, "Post foos").AssertOK(),
			Method(DELETE, "Delete foos").AssertOK(),
			Method(PUT, "Put foos").AssertOK(),
			Method(PATCH, "Patch foos").AssertOK(),
		),
		Endpoint("/foos", "Foos endpoint",
			Method(GET, "Get foos").AssertOK(),
			Method(POST, "Post foos").AssertOK(),
			Method(DELETE, "Delete foos").AssertOK(),
			Method(PUT, "Put foos").AssertOK(),
			Method(PATCH, "Patch foos").AssertOK(),
		),
	)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo_col", "bar_col", "baz_col"}).
		AddRow("foo1", 1, true).
		AddRow("foo2", 2, false).
		AddRow("foo3", 3, true))
	var coverage *Coverage
	s.Init(
		WithDatabase(db),
		WithCoverageCollect(func(cov *Coverage) {
			coverage = cov
		}),
		WithTesting(t),
		WithHttpDo(&dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo": "bar"}`),
		}),
		WithVar("OK", 201)).
		Run()
	require.NotNil(t, coverage)
	fmt.Printf("coverage: %+v\n", coverage)
	st, ok := coverage.Timings.Stats(false)
	require.True(t, ok)
	_ = st
	outliers := coverage.Timings.Outliers(0.99)
	require.Len(t, outliers, 1)
}

type dummyDo struct {
	status int
	body   []byte
}

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}
