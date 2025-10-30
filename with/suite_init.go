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

type SuiteInit interface {
	SetDb(db *sql.DB)
	SetDbArgMarkers(dbArgMarkers common.DatabaseArgMarkers)
	SetHttpDo(do common.HttpDo)
	SetApiHost(host string, port int)
	SetTesting(t *testing.T)
	SetVar(name string, value any)
	SetCookie(cookie *http.Cookie)
	SetReportCoverage(fn func(coverage *coverage.Coverage))
	SetCoverageCollector(collector coverage.Collector)
	SetOAS(r io.Reader)
	SetRepeats(n int, stopOnFailure bool, resets ...func())
	SetLogging(stdout io.Writer, stderr io.Writer)
	AddMockService(mock service.MockedService)
	AddSupportingImage(info ImageInfo)
}
