package with

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMockService(t *testing.T) {
	w := MockService("foo")
	require.Equal(t, Supporting, w.Stage())
	assert.Nil(t, w.Shutdown())

	mock := newMockInit()
	err := w.Init(mock)
	require.NoError(t, err)
	assert.Len(t, mock.called, 1)
	assert.Len(t, mock.services, 1)
	_, ok := mock.services["foo"]
	assert.True(t, ok)
	_, ok = mock.called["AddMockService:foo"]
	assert.True(t, ok)
	require.NotNil(t, w.Shutdown())
	w.Shutdown()()
}
