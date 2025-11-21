package with

import (
	"database/sql"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"io"
	"net/http"
	"os"
	"testing"
)

// Stage is the stage at which the With is executed during Suite initialisation
type Stage int

const (
	Initial    Stage = iota // for setting up e.g. vars (anything that just updates a setting in Suite)
	Supporting              // for spinning up supporting docker images (or running a Make on API image)
	Final                   // for final stage of Suite init, e.g. running the API container
)

// With is the interface that must be implemented by anything passed to Suite.Init()
type With interface {
	Init(init SuiteInit) error
	Stage() Stage
	Shutdown() func()
}

// Database initialises a marrow.Suite with an existing database (*sql.DB)
//
// The name arg is only needed when tests might use multiple different databases, otherwise an empty string is sufficient
func Database(name string, db *sql.DB, dbArgs common.DatabaseArgs) With {
	return withFn(func(init SuiteInit) {
		init.AddDb(name, db, dbArgs)
	})
}

// HttpDo initialises a marrow.Suite with an override for making http calls
func HttpDo(httpDo common.HttpDo) With {
	return withFn(func(init SuiteInit) {
		init.SetHttpDo(httpDo)
	})
}

// ApiHost initialises a marrow.Suite with a currently running API
//
// see also ApiImage for initialising a marrow.Suite with a docker image
func ApiHost(host string, port int) With {
	return withFn(func(init SuiteInit) {
		init.SetApiHost(host, port)
	})
}

// Testing initialises a marrow.Suite with a golang test
//
// When a marrow.Suite uses a testing.T, all tests are run with t.Run and api test pass/fail is indicated by the test runner
func Testing(t *testing.T) With {
	return withFn(func(init SuiteInit) {
		init.SetTesting(t)
	})
}

// Var initialises a marrow.Suite with a variable set
//
// Variables can be used in Endpoints and Methods to assert/require against
func Var(name string, value any) With {
	return withFn(func(init SuiteInit) {
		init.SetVar(name, value)
	})
}

// Cookie initialises a marrow.Suite with a pre-defined http.Cookie
//
// Any defined cookies can be used by requests made by a method
func Cookie(cookie *http.Cookie) With {
	return withFn(func(init SuiteInit) {
		init.SetCookie(cookie)
	})
}

// ReportCoverage initialises a marrow.Suite with a function that is called to report coverage
// after tests have been run
func ReportCoverage(fn func(coverage *coverage.Coverage)) With {
	return withFn(func(init SuiteInit) {
		init.SetReportCoverage(fn)
	})
}

// CoverageCollector initialises a marrow.Suite with a custom coverage collector
func CoverageCollector(collector coverage.Collector) With {
	return withFn(func(init SuiteInit) {
		init.SetCoverageCollector(collector)
	})
}

// OAS initialises a marrow.Suite with a reader for the OAS (Open API Specification) .yaml or .json
//
// When an OAS is provided, coverage can report test coverage against the spec
func OAS(r io.Reader) With {
	return withFn(func(init SuiteInit) {
		init.SetOAS(r)
	})
}

// Repeats initialises a marrow.Suite with a number of repeats to run
//
// repeats are run after the main endpoint+method tests - and is useful for gauging response timing stats
// in coverage for a larger number of calls
func Repeats(n int, stopOnFailure bool, resets ...func()) With {
	return withFn(func(init SuiteInit) {
		init.SetRepeats(n, stopOnFailure, resets...)
	})
}

// Logging initialises a marrow.Suite with log writers to use
//
// by default, the marrow.Suite will use os.Stdout and os.Stderr
//
// These log writers are not used if Testing is used
func Logging(stdout io.Writer, stderr io.Writer) With {
	return withFn(func(init SuiteInit) {
		init.SetLogging(stdout, stderr)
	})
}

// TraceTimings initialises a marrow.Suite to collect full trace timings on http calls for method tests
//
// if this is used, additional information about timings is available in coverage.Timings
func TraceTimings() With {
	return withFn(func(init SuiteInit) {
		init.SetTraceTimings(true)
	})
}

// DisableReaperShutdowns initialises a marrow.Suite to disable/enable container auto-shutdowns (RYUK)
//
// it sets the os env var "TESTCONTAINERS_RYUK_DISABLED"
func DisableReaperShutdowns(disable bool) With {
	const envRyukDisable = "TESTCONTAINERS_RYUK_DISABLED"
	return withFn(func(init SuiteInit) {
		if disable {
			_ = os.Setenv(envRyukDisable, "true")
		} else {
			_ = os.Setenv(envRyukDisable, "false")
		}
	})
}

type withFn func(init SuiteInit)

func (w withFn) Init(init SuiteInit) error {
	w(init)
	return nil
}

func (w withFn) Stage() Stage {
	return Initial
}

func (w withFn) Shutdown() func() {
	return nil
}
