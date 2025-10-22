package marrow

import (
	"database/sql"
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
	SetCoverageCollect(fn func(coverage *Coverage))
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

func WithCoverageCollect(fn func(coverage *Coverage)) With {
	return &with{
		fn: func(init SuiteInit) {
			init.SetCoverageCollect(fn)
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
