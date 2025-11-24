package marrow

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
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
		specF, err := os.Open("./_testdata/spec.yaml")
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
		useDo := &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":"bar"}`),
		}
		var buf bytes.Buffer
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").RequireOK(),
			),
		).Init(with.HttpDo(useDo), with.Logging(&buf, &buf), with.Repeats(2, true, func() {
			useDo.status = http.StatusNotFound
			useDo.body = []byte(`{}`)
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
	t.Run("endpoint fails", func(t *testing.T) {
		s := Suite(
			Endpoint("/foos", "",
				Method(GET, "").RequireNotFound(),
			),
		).Init(with.HttpDo(do))
		err := s.Run()
		require.NoError(t, err)
	})
	t.Run("initial with shutdown", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Initial, shutdown: func() {}}).Run()
		require.NoError(t, err)
		err = s.Run()
		require.NoError(t, err)
	})
	t.Run("initial with errors", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Initial, err: errors.New("fooey")}).Run()
		require.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
	})
	t.Run("supporting with shutdown", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Supporting, shutdown: func() {}}).Run()
		require.NoError(t, err)
		err = s.Run()
		require.NoError(t, err)
	})
	t.Run("supporting with errors", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Supporting, err: errors.New("fooey")}).Run()
		require.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
	})
	t.Run("final with shutdown", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Final, shutdown: func() {}}).Run()
		require.NoError(t, err)
		err = s.Run()
		require.NoError(t, err)
	})
	t.Run("final with errors", func(t *testing.T) {
		s := Suite()
		err := s.Init(&mockWith{stage: with.Final, err: errors.New("fooey")}).Run()
		require.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
	})
	t.Run("with mock services", func(t *testing.T) {
		s := Suite()
		err := s.Init(with.MockService("foo"), with.MockService("bar")).Run()
		require.NoError(t, err)
	})
	t.Run("with trace timings", func(t *testing.T) {
		s := Suite().Init(with.TraceTimings())
		err := s.Run()
		require.NoError(t, err)
		raw, ok := s.(*suite)
		require.True(t, ok)
		assert.True(t, raw.traceTimings)
	})
}

type errorReader struct{}

var _ io.Reader = (*errorReader)(nil)

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("error")
}

func TestWithDatabase(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := Suite().Init(with.Database("", db, common.DatabaseArgs{Style: common.NumberedDbArgs}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err = raw.runInits()
	require.NoError(t, err)
	ndb, ok := raw.dbs[""]
	assert.True(t, ok)
	assert.NotNil(t, ndb)
	assert.NotNil(t, ndb.db)
	assert.Equal(t, common.NumberedDbArgs, ndb.argMarkers.Style)
}

func TestWithHttpDo(t *testing.T) {
	s := Suite().Init(with.HttpDo(&dummyDo{}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.httpDo)
}

func TestWithApiHost(t *testing.T) {
	s := Suite().Init(with.ApiHost("localhost", 8080))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.Equal(t, "localhost", raw.host)
	assert.Equal(t, 8080, raw.port)
}

func TestWithTesting(t *testing.T) {
	s := Suite().Init(with.Testing(t))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.testing)
}

func TestWithVar(t *testing.T) {
	s := Suite().Init(with.Var("foo", "bar"))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.Len(t, raw.vars, 1)
}

func TestWithCookie(t *testing.T) {
	s := Suite().Init(with.Cookie(&http.Cookie{Name: "foo", Value: "bar"}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.Len(t, raw.cookies, 1)
	assert.Equal(t, "bar", raw.cookies["foo"].Value)
}

func TestWithReportCoverage(t *testing.T) {
	s := Suite().Init(with.ReportCoverage(func(coverage *coverage.Coverage) {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.reportCov)
}

func TestWithCoverageCollector(t *testing.T) {
	s := Suite().Init(with.CoverageCollector(coverage.NewNullCoverage()))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.covCollector)
}

func TestWithOAS(t *testing.T) {
	s := Suite().Init(with.OAS(strings.NewReader("")))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.oasReader)
}

func TestWithRepeats(t *testing.T) {
	s := Suite().Init(with.Repeats(10, true, func() {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.Equal(t, 10, raw.repeats)
	assert.True(t, raw.stopOnFailure)
	assert.Len(t, raw.repeatResets, 1)
}

func TestWithLogging(t *testing.T) {
	nw := &nullWriter{}
	s := Suite().Init(with.Logging(nw, nw))
	raw, ok := s.(*suite)
	require.True(t, ok)
	err := raw.runInits()
	require.NoError(t, err)
	assert.NotNil(t, raw.stdout)
	assert.Equal(t, nw, raw.stdout)
	assert.NotNil(t, raw.stderr)
	assert.Equal(t, nw, raw.stderr)
}

func TestAddSupportingImage(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		img := &mockImage{}
		s := Suite().Init(img, img)
		raw, ok := s.(*suite)
		require.True(t, ok)
		err := raw.runInits()
		require.NoError(t, err)
		assert.Len(t, raw.shutdowns, 2)
		assert.Len(t, raw.images, 2)
		assert.Nil(t, raw.apiImage)
		_, ok = raw.images["mock"]
		assert.True(t, ok)
		_, ok = raw.images["mock-2"]
		assert.True(t, ok)
	})
	t.Run("errors", func(t *testing.T) {
		img := &mockImage{err: errors.New("fooey")}
		s := Suite().Init(img, img)
		raw, ok := s.(*suite)
		require.True(t, ok)
		err := raw.runInits()
		require.Error(t, err)
		assert.Len(t, raw.shutdowns, 0)
		assert.True(t, img.shutdown)
	})
	t.Run("with api image", func(t *testing.T) {
		img := &mockApiImage{}
		s := Suite().Init(img)
		raw, ok := s.(*suite)
		require.True(t, ok)
		err := raw.runInits()
		require.NoError(t, err)
		assert.Len(t, raw.shutdowns, 1)
		assert.Len(t, raw.images, 0)
		assert.NotNil(t, raw.apiImage)
	})
}

func TestSuite_ResolveEnv(t *testing.T) {
	testCases := []struct {
		value     any
		vars      map[Var]any
		images    map[string]with.Image
		mocks     map[string]service.MockedService
		expect    string
		expectErr bool
		setup     func(t *testing.T) func()
	}{
		{
			value:  "foo",
			expect: "foo",
		},
		{
			value:  42,
			expect: "42",
		},
		{
			value:     Var("foo"),
			expectErr: true,
		},
		{
			value:  Var("foo"),
			vars:   map[Var]any{"foo": 42},
			expect: "42",
		},
		{
			value:     "{$foo}",
			expectErr: true,
		},
		{
			value:  "{$foo}",
			vars:   map[Var]any{"foo": 42},
			expect: "42",
		},
		{
			value:     "{$foo:host}",
			expectErr: true,
		},
		{
			value: "{$foo:host}",
			images: map[string]with.Image{
				"foo": &mockImage{},
			},
			expect: "localhost",
		},
		{
			value: "{$foo:port}",
			images: map[string]with.Image{
				"foo": &mockImage{},
			},
			expect: "8080",
		},
		{
			value: "{$foo:mport}",
			images: map[string]with.Image{
				"foo": &mockImage{},
			},
			expect: "50080",
		},
		{
			value: "{$foo:username}",
			images: map[string]with.Image{
				"foo": &mockImage{},
			},
			expect: "foo",
		},
		{
			value: "{$foo:password}",
			images: map[string]with.Image{
				"foo": &mockImage{},
			},
			expect: "bar",
		},
		{
			value: "{$foo:bar}",
			images: map[string]with.Image{
				"foo": &mockImage{
					envs: map[string]string{"bar": "baz"},
				},
			},
			expect: "baz",
		},
		{
			value: "{$foo:bar:baz}",
			images: map[string]with.Image{
				"foo": &mockImage{
					envs: map[string]string{"bar:baz": "buzz"},
				},
			},
			expect: "buzz",
		},
		{
			value:     "{$mock:foo:host}",
			expectErr: true,
		},
		{
			value: "{$mock:foo:host}",
			mocks: map[string]service.MockedService{
				"foo": &mockService{},
			},
			expect: "127.0.0.1",
		},
		{
			value: "{$mock:foo:port}",
			mocks: map[string]service.MockedService{
				"foo": &mockService{},
			},
			expect: "8888",
		},
		{
			value:  "{$foo",
			expect: "{$foo",
		},
		{
			value:  "\\\\\\{$foo}",
			expect: "\\\\{$foo}",
		},
		{
			value:     "{$env:TEST_AUTH}",
			expectErr: true,
		},
		{
			value:  "{$env:TEST_AUTH}",
			expect: "TEST_VALUE",
			setup: func(t *testing.T) func() {
				_ = os.Setenv("TEST_AUTH", "TEST_VALUE")
				return func() {
					_ = os.Unsetenv("TEST_AUTH")
				}
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			s := &suite{
				vars:         tc.vars,
				images:       tc.images,
				mockServices: tc.mocks,
			}
			if tc.setup != nil {
				td := tc.setup(t)
				if td != nil {
					defer td()
				}
			}
			result, err := s.ResolveEnv(tc.value)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expect, result)
			}
		})
	}
}
