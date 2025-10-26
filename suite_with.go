package marrow

import (
	"database/sql"
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
	SetDbArgMarkers(dbArgMarkers DatabaseArgMarkers)
	SetHttpDo(HttpDo)
	SetApiHost(host string, port int)
	SetApiImage(image string, more ...any) // how to set env etc.???
	SetTesting(t *testing.T)
	SetVar(name string, value any)
	SetCookie(cookie *http.Cookie)
	SetReportCoverage(fn func(coverage *coverage.Coverage))
	SetCoverageCollector(collector coverage.Collector)
	SetOAS(r io.Reader)
	SetRepeats(n int, stopOnFailure bool, resets ...func(si SuiteInit))
	// todo etc. more things that can be set/initialised prior to run
}

func WithDatabase(db *sql.DB) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetDb(db)
		}}
}

func WithDatabaseArgMarkers(dbArgMarkers DatabaseArgMarkers) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetDbArgMarkers(dbArgMarkers)
		}}
}

func WithHttpDo(httpDo HttpDo) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetHttpDo(httpDo)
		}}
}

func WithApiHost(host string, port int) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetApiHost(host, port)
		}}
}

func WithTesting(t *testing.T) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetTesting(t)
		}}
}

func WithVar(name string, value any) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetVar(name, value)
		}}
}

func WithCookie(cookie *http.Cookie) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetCookie(cookie)
		}}
}

func WithReportCoverage(fn func(coverage *coverage.Coverage)) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetReportCoverage(fn)
		}}
}

func WithOAS(r io.Reader) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetOAS(r)
		}}
}

func WithRepeats(n int, stopOnFailure bool, resets ...func(si SuiteInit)) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetRepeats(n, stopOnFailure, resets...)
		}}
}

type with struct {
	fn func(init SuiteInit)
}

func (w *with) Init(init SuiteInit) {
	w.fn(init)
}

type HttpDo interface {
	Do(req *http.Request) (*http.Response, error)
}
