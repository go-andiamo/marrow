package marrow

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/mocks/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSetVar(t *testing.T) {
	w := SetVar(After, "foo", "bar")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])
}

func TestClearVars(t *testing.T) {
	w := ClearVars(After)
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(map[Var]any{"foo": "bar"})
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.Empty(t, ctx.vars)
}

func TestDbInsert(t *testing.T) {
	w := DbInsert(After, "", "table_name", Columns{"foo": "bar"})
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestDbExec(t *testing.T) {
	w := DbExec(After, "", "DELETE FROM table_name")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestDbClearTable(t *testing.T) {
	w := DbClearTable(After, "", "table_name")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	defer db.Close()
	ctx := newTestContext(nil)
	ctx.dbs.register("", db, common.DatabaseArgs{})
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestMockServicesClearAll(t *testing.T) {
	w := MockServicesClearAll(After)
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.cleared)
}

func TestMockServiceClear(t *testing.T) {
	w := MockServiceClear(After, "mock")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.cleared)

	w = MockServiceClear(After, "unknown")
	err = w.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mock service ")
}

func TestMockServiceCall(t *testing.T) {
	w := MockServiceCall(After, "mock", "/foos", GET, http.StatusOK, nil)
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	ms := &mockMockedService{}
	ctx.mockServices["mock"] = ms
	err := w.Run(ctx)
	require.NoError(t, err)
	assert.True(t, ms.mocked)

	w = MockServiceCall(After, "unknown", "/foos", GET, http.StatusOK, nil)
	err = w.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mock service ")
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

func TestWait(t *testing.T) {
	w := Wait(After, 10)
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())

	ctx := newTestContext(nil)
	err := w.Run(ctx)
	require.NoError(t, err)
}
