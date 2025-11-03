package service

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestNewMockedService(t *testing.T) {
	svc := NewMockedService("foo")
	assert.Equal(t, "foo", svc.Name())
	assert.Equal(t, "localhost", svc.Host())
	assert.NotEqual(t, "127.0.0.1", svc.ActualHost())
	assert.NotEqual(t, "localhost", svc.ActualHost())
}

func TestMockedService_Start_MockCall_Shutdown(t *testing.T) {
	svc := NewMockedService("foo")
	err := svc.Start()
	require.NoError(t, err)

	assert.NotEmpty(t, svc.Host())
	assert.Greater(t, svc.Port(), 0)
	assert.Equal(t, fmt.Sprintf("http://%s:%d", svc.Host(), svc.Port()), svc.Url())

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/test", svc.Url()), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.False(t, svc.AssertCalled("/test", http.MethodGet))

	svc.MockCall("/test", http.MethodGet, http.StatusOK, []byte(`{"foo":"bar"}`), "X-Test-Hdr", "foo")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "foo", resp.Header.Get("X-Test-Hdr"))
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"foo":"bar"}`, string(data))
	assert.True(t, svc.AssertCalled("/test", http.MethodGet))
	// another call with the same request should be not found...
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// clear and add a couple...
	svc.Clear()
	svc.MockCall("/test", http.MethodGet, http.StatusOK, nil)
	svc.MockCall("/test", http.MethodGet, http.StatusOK, nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	svc.Shutdown()
}

func Test_bodyToBytes(t *testing.T) {
	testCases := []struct {
		body   any
		expect []byte
	}{
		{
			expect: []byte{},
		},
		{
			body:   []byte(`{"foo":"bar"}`),
			expect: []byte(`{"foo":"bar"}`),
		},
		{
			body:   json.RawMessage([]byte(`{"foo":"bar"}`)),
			expect: []byte(`{"foo":"bar"}`),
		},
		{
			body:   `{"foo":"bar"}`,
			expect: []byte(`{"foo":"bar"}`),
		},
		{
			body:   map[string]any{"foo": "bar"},
			expect: []byte(`{"foo":"bar"}`),
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			data := bodyToBytes(tc.body)
			assert.Equal(t, tc.expect, data)
		})
	}
}
