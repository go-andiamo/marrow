package with

import (
	"database/sql"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestWith_Initials(t *testing.T) {
	testCases := []any{
		Database("", nil, 0),
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
		TraceTimings(),
	}
	mock := newMockInit()
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			w, ok := tc.(With)
			require.True(t, ok)
			require.Equal(t, Initial, w.Stage())
			assert.Nil(t, w.Shutdown())
			err := w.Init(mock)
			require.NoError(t, err)
		})
	}
	assert.Len(t, mock.called, len(testCases))
	assert.Len(t, mock.called, 12)
}

func newMockInit() *mockInit {
	return &mockInit{
		called:   make(map[string]struct{}),
		services: make(map[string]service.MockedService),
		images:   make(map[string]Image),
	}
}

type mockInit struct {
	called   map[string]struct{}
	services map[string]service.MockedService
	images   map[string]Image
}

var _ SuiteInit = (*mockInit)(nil)

func (d *mockInit) AddDb(typeName string, db *sql.DB, dbArgMarkers common.DatabaseArgMarkers) {
	d.called["AddDb"] = struct{}{}
}

func (d *mockInit) SetHttpDo(do common.HttpDo) {
	d.called["SetHttpDo"] = struct{}{}
}

func (d *mockInit) SetApiHost(host string, port int) {
	d.called["SetApiHost"] = struct{}{}
}

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

func (d *mockInit) AddMockService(mock service.MockedService) {
	d.called["AddMockService:"+mock.Name()] = struct{}{}
	d.services[mock.Name()] = mock
}

func (d *mockInit) AddSupportingImage(info Image) {
	d.called["AddSupportingImage:"+info.Name()] = struct{}{}
	d.images[info.Name()] = info
}

func (d *mockInit) ResolveEnv(v any) (string, error) {
	return fmt.Sprintf("%v", v), nil
}

func (d *mockInit) SetTraceTimings(collect bool) {
	d.called["SetTraceTimings"] = struct{}{}
}
