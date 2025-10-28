package marrow

import (
	"bytes"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSuite_Run(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	t.Run("empty", func(t *testing.T) {
		s := Suite()
		err := s.Run()
		require.NoError(t, err)
	})
	t.Run("empty with vars", func(t *testing.T) {
		s := Suite().Init(with.Var("foo", "bar"))
		err := s.Run()
		require.NoError(t, err)
	})
	t.Run("empty with cookies", func(t *testing.T) {
		s := Suite().Init(with.Cookie(&http.Cookie{Name: "foo", Value: "bar"}))
		err := s.Run()
		require.NoError(t, err)
	})
	t.Run("success with single endpoint", func(t *testing.T) {
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do))
		err := s.Run()
		require.NoError(t, err)
	})
	t.Run("with coverage collector", func(t *testing.T) {
		cov := coverage.NewCoverage()
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.CoverageCollector(cov))
		err := s.Run()
		require.NoError(t, err)
		assert.Len(t, cov.Timings, 1)
	})
	t.Run("with report coverage", func(t *testing.T) {
		var cov *coverage.Coverage
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}))
		err := s.Run()
		require.NoError(t, err)
		assert.Len(t, cov.Timings, 1)
	})
	t.Run("errors with coverage collector & report coverage", func(t *testing.T) {
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.CoverageCollector(coverage.NewNullCoverage()), with.ReportCoverage(func(coverage *coverage.Coverage) {}))
		err := s.Run()
		require.Error(t, err)
		assert.Equal(t, "cannot report coverage with custom coverage collector", err.Error())
	})
	t.Run("with OAS & coverage", func(t *testing.T) {
		specF, err := os.Open("./_examples/petstore.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer specF.Close()
		var cov *coverage.Coverage
		s := Suite(
			Endpoint("/api/pets", "",
				Method(GET, "").AssertOK(),
			),
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.OAS(specF), with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}))
		err = s.Run()
		require.NoError(t, err)
		assert.Len(t, cov.Timings, 2)
		specCov, err := cov.SpecCoverage()
		require.NoError(t, err)
		tot, covd, perc := specCov.PathsCovered()
		assert.Equal(t, 5, tot)
		assert.Equal(t, 1, covd)
		assert.Equal(t, 0.2, perc)
		assert.Len(t, specCov.UnknownPaths, 1)
	})
	t.Run("errors with OAS reader error", func(t *testing.T) {
		s := Suite(
			Endpoint("/api/pets", "",
				Method(GET, "").AssertOK(),
			),
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.OAS(&errorReader{}), with.ReportCoverage(func(coverage *coverage.Coverage) {}))
		err := s.Run()
		require.Error(t, err)
	})
	t.Run("with logging", func(t *testing.T) {
		var buf bytes.Buffer
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.Logging(&buf, &buf))
		err := s.Run()
		require.NoError(t, err)
		assert.Equal(t, 3, strings.Count(buf.String(), "=== RUN   "))
		assert.Equal(t, 3, strings.Count(buf.String(), "--- PASS: "))
		assert.Contains(t, buf.String(), "//foos\n")
		assert.Contains(t, buf.String(), "//foos/GET\n")
		assert.Contains(t, buf.String(), "//foos (")
		assert.Contains(t, buf.String(), "//foos/GET (")
	})
	t.Run("with repeats", func(t *testing.T) {
		do := &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":"bar"}`),
		}
		var buf bytes.Buffer
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.Logging(&buf, &buf), with.Repeats(2, false))
		err := s.Run()
		require.NoError(t, err)
		assert.Equal(t, 3, strings.Count(buf.String(), "=== RUN   "))
		assert.Equal(t, 3, strings.Count(buf.String(), "--- PASS: "))
		assert.Contains(t, buf.String(), "//foos\n")
		assert.Contains(t, buf.String(), "//foos/GET\n")
		assert.Contains(t, buf.String(), "//foos (")
		assert.Contains(t, buf.String(), "//foos/GET (")
		assert.Contains(t, buf.String(), "\n>>> REPEAT 1/2")
		assert.Contains(t, buf.String(), "\n>>> REPEAT 2/2")
		assert.Equal(t, 2, strings.Count(buf.String(), "\n    FINISHED ("))
	})
	t.Run("with repeats - stop on fail", func(t *testing.T) {
		do := &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":"bar"}`),
		}
		var buf bytes.Buffer
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").AssertOK(),
			),
		).Init(with.HttpDo(do), with.Logging(&buf, &buf), with.Repeats(2, true, func() {
			do.status = http.StatusNotFound
			do.body = []byte(`{}`)
		}))
		err := s.Run()
		require.NoError(t, err)
		assert.Equal(t, 3, strings.Count(buf.String(), "=== RUN   "))
		assert.Equal(t, 3, strings.Count(buf.String(), "--- PASS: "))
		assert.Contains(t, buf.String(), "//foos\n")
		assert.Contains(t, buf.String(), "//foos/GET\n")
		assert.Contains(t, buf.String(), "//foos (")
		assert.Contains(t, buf.String(), "//foos/GET (")
		assert.Contains(t, buf.String(), "\n>>> REPEAT 1/2")
		assert.Contains(t, buf.String(), "\n    FAILED (")
	})
}

type errorReader struct{}

var _ io.Reader = (*errorReader)(nil)

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("error")
}

/*
func TestSuite_Demo(t *testing.T) {
	t.Skip()
	specF, err := os.Open("./_examples/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer specF.Close()
	sub := Endpoint("/bars/{id}", "Bars",
		Method(GET, "Get bars").
			PathParam(Var("bar_id")).
			AssertStatus(Var("OK")),
	)
	s := Suite(
		Endpoint("/api/pets", "Pets",
			Method(GET, "Get pets").AssertOK(),
			Endpoint("/{pet_id}", "Get pet",
				Method(GET, "Get pet").
					PathParam(12345).
					AssertOK(),
			),
			Method(DELETE, "Delete pets").AssertOK(),
		),
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
				AssertEqual(JsonPath(Var("body"), "foo"), 123.1),
			Method(POST, "Post foos").AssertOK().
				SetVar(After, "bar_id", "1234"),
			Method(DELETE, "Delete foos").AssertOK(),
			Method(PUT, "Put foos").AssertOK(),
			Method(PATCH, "Patch foos").AssertOK(),
			sub,
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
	var cov *coverage.Coverage
	s.Init(
		with.OAS(specF),
		with.Database(db),
		with.ReportCoverage(func(c *coverage.Coverage) {
			cov = c
		}),
		with.Testing(t),
		with.Repeats(10, false, func() {
			// reset the mock db...
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"foo_col", "bar_col", "baz_col"}).
				AddRow("foo1", 1, true).
				AddRow("foo2", 2, false).
				AddRow("foo3", 3, true))
		}),
		with.HttpDo(&dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo": "bar"}`),
		}),
		with.Var("OK", 201)).
		Run()

	require.NotNil(t, cov)
	stats, ok := cov.Timings.Stats(false)
	require.True(t, ok)
	assert.Equal(t, 14, stats.Count)
	assert.Less(t, stats.Variance, 0.01)
	outliers := cov.Timings.Outliers(0.99)
	assert.Len(t, outliers, 1)

	specCov, err := cov.SpecCoverage()
	require.NoError(t, err)
	tot, covd, perc := specCov.PathsCovered()
	t.Logf("Spec Coverage Paths:\n\tTotal: %d, Covered: %d, Perc: %.1f%%\n", tot, covd, perc*100)
	tot, covd, perc = specCov.MethodsCovered()
	t.Logf("Spec Coverage Methods:\n\tTotal: %d, Covered: %d, Perc: %.1f%%\n", tot, covd, perc*100)
}
*/

func TestWithDatabase(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := Suite().Init(with.Database(db))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.db)
}

func TestWithDatabaseArgMarkers(t *testing.T) {
	s := Suite().Init(with.DatabaseArgMarkers(common.NumberedDbArgs))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, common.NumberedDbArgs, raw.dbArgMarkers)
}

func TestWithHttpDo(t *testing.T) {
	s := Suite().Init(with.HttpDo(&dummyDo{}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.httpDo)
}

func TestWithApiHost(t *testing.T) {
	s := Suite().Init(with.ApiHost("localhost", 8080))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, "localhost", raw.host)
	assert.Equal(t, 8080, raw.port)
}

func TestWithTesting(t *testing.T) {
	s := Suite().Init(with.Testing(t))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.testing)
}

func TestWithVar(t *testing.T) {
	s := Suite().Init(with.Var("foo", "bar"))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Len(t, raw.vars, 1)
}

func TestWithCookie(t *testing.T) {
	s := Suite().Init(with.Cookie(&http.Cookie{Name: "foo", Value: "bar"}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Len(t, raw.cookies, 1)
	assert.Equal(t, "bar", raw.cookies["foo"].Value)
}

func TestWithReportCoverage(t *testing.T) {
	s := Suite().Init(with.ReportCoverage(func(coverage *coverage.Coverage) {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.reportCov)
}

func TestWithCoverageCollector(t *testing.T) {
	s := Suite().Init(with.CoverageCollector(coverage.NewNullCoverage()))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.covCollector)
}

func TestWithOAS(t *testing.T) {
	s := Suite().Init(with.OAS(strings.NewReader("")))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.oasReader)
}

func TestWithRepeats(t *testing.T) {
	s := Suite().Init(with.Repeats(10, true, func() {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, 10, raw.repeats)
	assert.True(t, raw.stopOnFailure)
	assert.Len(t, raw.repeatResets, 1)
}

func TestWithLogging(t *testing.T) {
	nw := &nullWriter{}
	s := Suite().Init(with.Logging(nw, nw))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.stdout)
	assert.Equal(t, nw, raw.stdout)
	assert.NotNil(t, raw.stderr)
	assert.Equal(t, nw, raw.stderr)
}

type nullWriter struct{}

var _ io.Writer = (*nullWriter)(nil)

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type dummyDo struct {
	status int
	body   []byte
	err    error
}

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}
