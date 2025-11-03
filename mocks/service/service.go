package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type MockedService interface {
	Name() string
	Host() string
	ActualHost() string
	Port() int
	Url() string
	Start() error
	Shutdown()
	Clear()
	MockCall(path string, method string, responseStatus int, responseBody any, headers ...string)
	AssertCalled(path string, method string) bool
}

func NewMockedService(name string) MockedService {
	return &mockedService{
		name:       name,
		host:       "localhost",
		actualHost: localIP(),
		endpoints:  make(map[string]*mockedEndpoint),
	}
}

type mockedService struct {
	name       string
	host       string
	actualHost string
	port       int
	server     *http.Server
	listener   net.Listener
	mu         sync.RWMutex
	endpoints  map[string]*mockedEndpoint
}

var _ MockedService = &mockedService{}

type mockedEndpoint struct {
	calls    int
	statuses []int
	headers  []map[string]string
	bodies   [][]byte
}

var _ MockedService = (*mockedService)(nil)
var _ http.Handler = (*mockedService)(nil)

func (m *mockedService) Name() string {
	return m.name
}

func (m *mockedService) Host() string {
	return m.host
}

func (m *mockedService) ActualHost() string {
	return m.actualHost
}

func (m *mockedService) Port() int {
	return m.port
}

func (m *mockedService) Url() string {
	return "http://" + m.host + ":" + strconv.Itoa(m.port)
}

func (m *mockedService) Start() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mocked service: %w", err)
		}
	}()
	// listen on "127.0.0.1:0" (i.e. port 0) tells the OS to pick an unused port
	if m.listener, err = net.Listen("tcp", "127.0.0.1:0"); err == nil {
		addr := m.listener.Addr().(*net.TCPAddr)
		m.port = addr.Port
		m.server = &http.Server{Handler: m}
		go func() {
			_ = m.server.Serve(m.listener)
		}()
	}
	return
}

func (m *mockedService) Shutdown() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.server.Shutdown(ctx)
	}
}

func (m *mockedService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpoints = make(map[string]*mockedEndpoint)
}

func (m *mockedService) MockCall(path string, method string, responseStatus int, responseBody any, headers ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l := len(headers)
	hdrs := make(map[string]string, l/2)
	for i := 0; i < l; i += 2 {
		hdrs[headers[i]] = headers[i+1]
	}
	if ep, ok := m.endpoints[method+" "+path]; ok {
		ep.statuses = append(ep.statuses, responseStatus)
		ep.bodies = append(ep.bodies, bodyToBytes(responseBody))
		ep.headers = append(ep.headers, hdrs)
	} else {
		m.endpoints[method+" "+path] = &mockedEndpoint{
			statuses: []int{responseStatus},
			bodies:   [][]byte{bodyToBytes(responseBody)},
			headers:  []map[string]string{hdrs},
		}
	}
}

func bodyToBytes(body any) []byte {
	result := make([]byte, 0)
	if body != nil {
		switch bt := body.(type) {
		case json.RawMessage:
			return bt
		case []byte:
			return bt
		case string:
			return []byte(bt)
		default:
			if data, err := json.Marshal(body); err == nil {
				return data
			}
		}
	}
	return result
}

func (m *mockedService) AssertCalled(path string, method string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ep, ok := m.endpoints[method+" "+path]; ok && ep.calls > 0 {
		return true
	}
	return false
}

func (m *mockedService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ep, ok := m.endpoints[r.Method+" "+r.URL.Path]; ok && ep.calls < len(ep.statuses) {
		n := ep.calls
		ep.calls++
		hdrs := ep.headers[n]
		seenContentType := false
		for k, v := range hdrs {
			w.Header().Set(k, v)
			seenContentType = seenContentType || k == "Content-Type"
		}
		if !seenContentType {
			w.Header().Add("Content-Type", "application/json")
		}
		w.WriteHeader(ep.statuses[n])
		_, _ = w.Write(ep.bodies[n])
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func localIP() (result string) {
	result = "127.0.0.1"
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			// check if the address is an IP address and not a loopback...
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil { // ensure it's an IPv4 address
					result = ipNet.IP.String()
					break
				}
			}
		}
	}
	return result
}
