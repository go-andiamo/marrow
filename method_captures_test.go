package marrow

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestMethod_Authorize(t *testing.T) {
	m := Method(GET, "").Authorize(func(ctx Context) error {
		return nil
	})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.authFns, 1)
}

func TestMethod_Capture(t *testing.T) {
	m := Method(GET, "").
		Capture(SetVar(Before, "foo", "bar")).
		Capture(SetVar(After, "foo", "bar"))
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_Do(t *testing.T) {
	m := Method(GET, "").
		Do(SetVar(Before, "foo", "bar"), nil, SetVar(After, "foo", "bar"))
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_CaptureFunc(t *testing.T) {
	m := Method(GET, "").
		CaptureFunc(Before, func(c Context) error {
			return nil
		}).
		CaptureFunc(After, func(ctx Context) error {
			return nil
		})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_DoFunc(t *testing.T) {
	fn := func(ctx Context) error {
		return nil
	}
	m := Method(GET, "").
		DoFunc(Before, fn, nil, fn).
		DoFunc(After, fn, nil, fn)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 2)
	assert.Len(t, raw.postCaptures, 2)
	assert.Len(t, raw.postOps, 2)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
	assert.False(t, raw.postOps[1].isExpectation)
	assert.Equal(t, 1, raw.postOps[1].index)
}

func TestMethod_SetVar(t *testing.T) {
	m := Method(GET, "").
		SetVar(Before, "foo", "bar").
		SetVar(After, "foo", "bar")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_ClearVars(t *testing.T) {
	m := Method(GET, "").
		ClearVars(Before).
		ClearVars(After)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_DbInsert(t *testing.T) {
	m := Method(GET, "").
		DbInsert(Before, "", "table", Columns{}).
		DbInsert(After, "", "table", Columns{})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_DbExec(t *testing.T) {
	m := Method(GET, "").
		DbExec(Before, "", "DELETE FROM tble").
		DbExec(After, "", "DELETE FROM tble")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_DbClearTables(t *testing.T) {
	m := Method(GET, "").
		DbClearTables(Before, "", "table_a", "table_b").
		DbClearTables(After, "", "table_a", "table_b")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 2)
	assert.Len(t, raw.postCaptures, 2)
	assert.Len(t, raw.postOps, 2)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
	assert.False(t, raw.postOps[1].isExpectation)
	assert.Equal(t, 1, raw.postOps[1].index)
}

func TestMethod_SetCookie(t *testing.T) {
	m := Method(GET, "").
		SetCookie(&http.Cookie{})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
}

func TestMethod_StoreCookie(t *testing.T) {
	m := Method(GET, "").
		StoreCookie("foo")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_MockServicesClearAll(t *testing.T) {
	m := Method(GET, "").
		MockServicesClearAll(Before).
		MockServicesClearAll(After)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_MockServiceClear(t *testing.T) {
	m := Method(GET, "").
		MockServiceClear(Before, "mock").
		MockServiceClear(After, "mock")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_MockServiceCall(t *testing.T) {
	m := Method(GET, "").
		MockServiceCall("mock", "/foos", GET, 200, nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
}

func TestMethod_Wait(t *testing.T) {
	m := Method(GET, "").
		Wait(Before, 10).
		Wait(After, 10)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.preCaptures, 1)
	assert.Len(t, raw.postCaptures, 1)
	assert.Len(t, raw.postOps, 1)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}
