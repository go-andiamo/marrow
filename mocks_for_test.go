package marrow

import (
	"bytes"
	"crypto/tls"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/mocks/service"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
)

type mockWith struct {
	stage    with.Stage
	err      error
	shutdown func()
}

var _ with.With = (*mockWith)(nil)

func (m mockWith) Init(init with.SuiteInit) error {
	return m.err
}

func (m mockWith) Stage() with.Stage {
	return m.stage
}

func (m mockWith) Shutdown() func() {
	return m.shutdown
}

type mockRunnable struct {
	err       error
	reportErr error
}

var _ Runnable = (*mockRunnable)(nil)

func (m *mockRunnable) Run(ctx Context) error {
	if m.err != nil {
		return m.err
	} else if m.reportErr != nil {
		ctx.reportFailure(m.reportErr)
	}
	return nil
}

func (m *mockRunnable) Frame() *framing.Frame {
	return framing.NewFrame(0)
}

type mockImage struct {
	err      error
	shutdown bool
	envs     map[string]string
}

var _ with.With = (*mockImage)(nil)
var _ with.Image = (*mockImage)(nil)
var _ with.ImageResolveEnv = (*mockImage)(nil)

func (m *mockImage) Init(init with.SuiteInit) error {
	init.AddSupportingImage(m)
	return m.err
}

func (m *mockImage) Stage() with.Stage {
	return with.Supporting
}

func (m *mockImage) Shutdown() func() {
	return func() {
		m.shutdown = true
	}
}

func (m *mockImage) Name() string {
	return "mock"
}

func (m *mockImage) Host() string {
	return "localhost"
}

func (m *mockImage) Port() string {
	return "8080"
}

func (m *mockImage) MappedPort() string {
	return "50080"
}

func (m *mockImage) IsDocker() bool {
	return true
}

func (m *mockImage) Username() string {
	return "foo"
}

func (m *mockImage) Password() string {
	return "bar"
}

func (m *mockImage) ResolveEnv(tokens ...string) (string, bool) {
	s, ok := m.envs[strings.Join(tokens, ":")]
	return s, ok
}

type mockApiImage struct {
	mockImage
}

var _ with.ImageApi = (*mockApiImage)(nil)

func (m *mockApiImage) Init(init with.SuiteInit) error {
	init.SetApiHost("localhost", 8080)
	init.AddSupportingImage(m)
	return m.err
}

func (m *mockApiImage) Name() string {
	return "api"
}

func (m *mockApiImage) Stage() with.Stage {
	return with.Final
}

func (m *mockApiImage) Container() testcontainers.Container {
	return nil
}

func (m *mockApiImage) IsApi() bool {
	return true
}

type mockService struct{}

var _ service.MockedService = (*mockService)(nil)

func (m *mockService) Name() string {
	return "mock"
}

func (m *mockService) Host() string {
	return "localhost"
}

func (m *mockService) ActualHost() string {
	return "127.0.0.1"
}

func (m *mockService) Port() int {
	return 8888
}

func (m *mockService) Url() string {
	return ""
}

func (m *mockService) Start() error {
	return nil
}

func (m *mockService) Shutdown() {
	// mock does nothing
}

func (m *mockService) Clear() {
	// mock does nothing
}

func (m *mockService) MockCall(path string, method string, responseStatus int, responseBody any, headers ...string) {
	// mock does nothing
}

func (m *mockService) AssertCalled(path string, method string) bool {
	return false
}

type nullWriter struct{}

var _ io.Writer = (*nullWriter)(nil)

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type dummyDo struct {
	status int
	body   []byte
	err    error
}

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	if tt := httptrace.ContextClientTrace(req.Context()); tt != nil {
		if tt.DNSStart != nil {
			tt.DNSStart(httptrace.DNSStartInfo{})
		}
		if tt.DNSDone != nil {
			tt.DNSDone(httptrace.DNSDoneInfo{})
		}
		if tt.ConnectStart != nil {
			tt.ConnectStart("", "")
		}
		if tt.ConnectDone != nil {
			tt.ConnectDone("", "", nil)
		}
		if tt.TLSHandshakeStart != nil {
			tt.TLSHandshakeStart()
		}
		if tt.TLSHandshakeDone != nil {
			tt.TLSHandshakeDone(tls.ConnectionState{}, nil)
		}
		if tt.GotConn != nil {
			tt.GotConn(httptrace.GotConnInfo{})
		}
		if tt.WroteRequest != nil {
			tt.WroteRequest(httptrace.WroteRequestInfo{})
		}
		if tt.GotFirstResponseByte != nil {
			tt.GotFirstResponseByte()
		}
	}
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}

type mockMockedService struct {
	called  bool
	cleared bool
	mocked  bool
}

var _ service.MockedService = (*mockMockedService)(nil)

func (m *mockMockedService) Name() string {
	return "mock"
}

func (m *mockMockedService) Host() string {
	return "localhost"
}

func (m *mockMockedService) ActualHost() string {
	return "127.0.0.1"
}

func (m *mockMockedService) Port() int {
	return 8080
}

func (m *mockMockedService) Url() string {
	return "http://localhost:8080"
}

func (m *mockMockedService) Start() error {
	// does nothing
	return nil
}

func (m *mockMockedService) Shutdown() {
	// does nothing
}

func (m *mockMockedService) Clear() {
	m.cleared = true
}

func (m *mockMockedService) MockCall(path string, method string, responseStatus int, responseBody any, headers ...string) {
	m.mocked = true
}

func (m *mockMockedService) AssertCalled(path string, method string) bool {
	return m.called
}
