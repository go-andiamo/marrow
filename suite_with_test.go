package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWithDatabase(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := Suite().Init(WithDatabase(db))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.db)
}

func TestWithDatabaseArgMarkers(t *testing.T) {
	s := Suite().Init(WithDatabaseArgMarkers(NumberedDbArgs))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, NumberedDbArgs, raw.dbArgMarkers)
}

func TestWithHttpDo(t *testing.T) {
	s := Suite().Init(WithHttpDo(&dummyDo{}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.httpDo)
}

func TestWithApiHost(t *testing.T) {
	s := Suite().Init(WithApiHost("localhost", 8080))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, "localhost", raw.host)
	assert.Equal(t, 8080, raw.port)
}

func TestWithTesting(t *testing.T) {
	s := Suite().Init(WithTesting(t))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.testing)
}

func TestWithVar(t *testing.T) {
	s := Suite().Init(WithVar("foo", "bar"))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Len(t, raw.vars, 1)
}

func TestWithCookie(t *testing.T) {
	s := Suite().Init(WithCookie(&http.Cookie{Name: "foo", Value: "bar"}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Len(t, raw.cookies, 1)
	assert.Equal(t, "bar", raw.cookies["foo"].Value)
}

func TestWithReportCoverage(t *testing.T) {
	s := Suite().Init(WithReportCoverage(func(coverage *coverage.Coverage) {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.reportCov)
}

func TestWithCoverageCollector(t *testing.T) {
	s := Suite().Init(WithCoverageCollector(coverage.NewNullCoverage()))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.covCollector)
}

func TestWithOAS(t *testing.T) {
	s := Suite().Init(WithOAS(strings.NewReader("")))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.NotNil(t, raw.oasReader)
}

func TestWithRepeats(t *testing.T) {
	s := Suite().Init(WithRepeats(10, true, func(si SuiteInit) {}))
	raw, ok := s.(*suite)
	require.True(t, ok)
	raw.runInits()
	assert.Equal(t, 10, raw.repeats)
	assert.True(t, raw.stopOnFailure)
	assert.Len(t, raw.repeatResets, 1)
}

func TestWithLogging(t *testing.T) {
	nw := &nullWriter{}
	s := Suite().Init(WithLogging(nw, nw))
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
