package with

import (
	"database/sql"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"io"
	"net/http"
	"testing"
)

type With interface {
	Init(init SuiteInit)
}

type SuiteInit interface {
	SetDb(db *sql.DB)
	SetDbArgMarkers(dbArgMarkers common.DatabaseArgMarkers)
	SetHttpDo(do common.HttpDo)
	SetApiHost(host string, port int)
	SetApiImage(image string, more ...any) // how to set env etc.???
	SetTesting(t *testing.T)
	SetVar(name string, value any)
	SetCookie(cookie *http.Cookie)
	SetReportCoverage(fn func(coverage *coverage.Coverage))
	SetCoverageCollector(collector coverage.Collector)
	SetOAS(r io.Reader)
	SetRepeats(n int, stopOnFailure bool, resets ...func())
	SetLogging(stdout io.Writer, stderr io.Writer)
}

func Database(db *sql.DB) With {
	return withFn(func(init SuiteInit) {
		init.SetDb(db)
	})
}

func DatabaseArgMarkers(dbArgMarkers common.DatabaseArgMarkers) With {
	return withFn(func(init SuiteInit) {
		init.SetDbArgMarkers(dbArgMarkers)
	})
}

func HttpDo(httpDo common.HttpDo) With {
	return withFn(func(init SuiteInit) {
		init.SetHttpDo(httpDo)
	})
}

func ApiHost(host string, port int) With {
	return withFn(func(init SuiteInit) {
		init.SetApiHost(host, port)
	})
}

func Testing(t *testing.T) With {
	return withFn(func(init SuiteInit) {
		init.SetTesting(t)
	})
}

func Var(name string, value any) With {
	return withFn(func(init SuiteInit) {
		init.SetVar(name, value)
	})
}

func Cookie(cookie *http.Cookie) With {
	return withFn(func(init SuiteInit) {
		init.SetCookie(cookie)
	})
}

func ReportCoverage(fn func(coverage *coverage.Coverage)) With {
	return withFn(func(init SuiteInit) {
		init.SetReportCoverage(fn)
	})
}

func CoverageCollector(collector coverage.Collector) With {
	return withFn(func(init SuiteInit) {
		init.SetCoverageCollector(collector)
	})
}

func OAS(r io.Reader) With {
	return withFn(func(init SuiteInit) {
		init.SetOAS(r)
	})
}

func Repeats(n int, stopOnFailure bool, resets ...func()) With {
	return withFn(func(init SuiteInit) {
		init.SetRepeats(n, stopOnFailure, resets...)
	})
}

func Logging(stdout io.Writer, stderr io.Writer) With {
	return withFn(func(init SuiteInit) {
		init.SetLogging(stdout, stderr)
	})
}

type withFn func(init SuiteInit)

func (w withFn) Init(init SuiteInit) {
	w(init)
}
