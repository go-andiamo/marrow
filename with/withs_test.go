package with

import (
	"database/sql"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestWithInterface(t *testing.T) {
	testCases := []any{
		Database(nil),
		DatabaseArgMarkers(0),
		HttpDo(nil),
		ApiHost("", 0),
		Testing(nil),
		Var("", nil),
		Cookie(nil),
		ReportCoverage(nil),
		CoverageCollector(nil),
		OAS(nil),
		Repeats(0, false),
		Logging(nil, nil),
	}
	mock := &mockInit{called: make(map[string]struct{})}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			w, ok := tc.(With)
			require.True(t, ok)
			w.Init(mock)
		})
	}
	assert.Len(t, mock.called, len(testCases))
	assert.Len(t, mock.called, 12) // should be more eventually
}

type mockInit struct {
	called map[string]struct{}
}

var _ SuiteInit = (*mockInit)(nil)

func (d *mockInit) SetDb(db *sql.DB) {
	d.called["SetDb"] = struct{}{}
}

func (d *mockInit) SetDbArgMarkers(dbArgMarkers common.DatabaseArgMarkers) {
	d.called["SetDbArgMarkers"] = struct{}{}
}

func (d *mockInit) SetHttpDo(do common.HttpDo) {
	d.called["SetHttpDo"] = struct{}{}
}

func (d *mockInit) SetApiHost(host string, port int) {
	d.called["SetApiHost"] = struct{}{}
}

/*
func (d *mockInit) SetApiImage(image string, more ...any) {
	d.called["SetApiImage"] = struct{}{}
}
*/

func (d *mockInit) SetTesting(t *testing.T) {
	d.called["SetTesting"] = struct{}{}
}

func (d *mockInit) SetVar(name string, value any) {
	d.called["SetVar"] = struct{}{}
}

func (d *mockInit) SetCookie(cookie *http.Cookie) {
	d.called["SetCookie"] = struct{}{}
}

func (d *mockInit) SetReportCoverage(fn func(coverage *coverage.Coverage)) {
	d.called["SetReportCoverage"] = struct{}{}
}

func (d *mockInit) SetCoverageCollector(collector coverage.Collector) {
	d.called["SetCoverageCollector"] = struct{}{}
}

func (d *mockInit) SetOAS(r io.Reader) {
	d.called["SetOAS"] = struct{}{}
}

func (d *mockInit) SetRepeats(n int, stopOnFailure bool, resets ...func()) {
	d.called["SetRepeats"] = struct{}{}
}

func (d *mockInit) SetLogging(stdout io.Writer, stderr io.Writer) {
	d.called["SetLogging"] = struct{}{}
}
