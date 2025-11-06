package with

import (
	"database/sql"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
	"io"
	"net/http"
	"testing"
)

// SuiteInit is the interface passed to With.Init
// and allows the With to set things in the marrow.Suite prior to tests run
//
// Within one With.Init, it can make one or more calls to this interface.
// For example, a With that spins up a supporting database docker container might call
// both SuiteInit.AddSupportingImage and SuiteInit.AddDb
type SuiteInit interface {
	// AddDb adds a supporting database to the marrow.Suite
	//
	// The dbName is normally the flavour of db (e.g. "mysql")
	//
	// If the tests only use one supporting database, dbName can just be ""
	//
	// see also Database
	AddDb(dbName string, db *sql.DB, dbArgs common.DatabaseArgs)
	// SetApiHost sets the host and port for the API being tested
	//
	// This can be called multiple times but only one API is used - therefore, last one wins
	//
	// see also ApiHost
	SetApiHost(host string, port int)
	// SetTesting tells the marrow.Suite to run tests in the provided go test
	//
	// If this is not set, the marrow.Suite uses its own internal test runner
	//
	// see also Testing
	SetTesting(t *testing.T)
	// SetVar sets an initial variable in the marrow.Suite which is initialised in the marrow.Context on tests run
	//
	// see also Var
	SetVar(name string, value any)
	// SetCookie sets an initial cookie in the marrow.Suite which is initialised in the marrow.Context on tests run
	//
	// see also Cookie
	SetCookie(cookie *http.Cookie)
	// SetReportCoverage provides a callback function to receive test coverage.Coverage
	//
	// if this callback is not provided, the marrow.Suite will not collect coverage information
	//
	// see also ReportCoverage
	SetReportCoverage(fn func(coverage *coverage.Coverage))
	// SetCoverageCollector sets a custom coverage collector in the marrow.Suite
	//
	// see also CoverageCollector
	SetCoverageCollector(collector coverage.Collector)
	// SetOAS sets an Open API Spec reader (json or yaml)
	//
	// When an OAS is provided, coverage can report test coverage against the spec
	//
	// see also OAS
	SetOAS(r io.Reader)
	// SetRepeats initialises a marrow.Suite with a number of repeats to run
	//
	// repeats are run after the main endpoint+method tests - and is useful for gauging response timing stats
	// in coverage for a larger number of calls
	//
	// see also Repeats
	SetRepeats(n int, stopOnFailure bool, resets ...func())
	// SetLogging initialises a marrow.Suite with log writers to use
	//
	// by default, the marrow.Suite will use os.Stdout and os.Stderr
	//
	// These log writers are not used if Testing is used
	SetLogging(stdout io.Writer, stderr io.Writer)
	// AddMockService adds a mock service for use in tests
	AddMockService(mock service.MockedService)
	// AddSupportingImage adds a supporting docker container image used by the tests
	AddSupportingImage(info Image)
	// ResolveEnv is used to resolve environment variables
	//
	// if the value passed is a string, it can contain special markers - see ApiImage
	ResolveEnv(v any) (string, error)
	// SetHttpDo sets the marrow.Suite with an override for making http calls
	//
	// by default, the marrow.Suite will ue http.DefaultClient
	//
	// see also HttpDo
	SetHttpDo(do common.HttpDo)
	// SetTraceTimings sets whether the marrow.Suite should collect trace timings within coverage
	//
	// see also TraceTimings
	SetTraceTimings(collect bool)
}
