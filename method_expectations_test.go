package marrow

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethod_Expect(t *testing.T) {
	m := Method(GET, "").Expect(ExpectationFunc(func(ctx Context) (unmet error, err error) {
		return nil, nil
	}))
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertOK(t *testing.T) {
	m := Method(GET, "").AssertOK()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertCreated(t *testing.T) {
	m := Method(GET, "").AssertCreated()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertAccepted(t *testing.T) {
	m := Method(GET, "").AssertAccepted()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNoContent(t *testing.T) {
	m := Method(GET, "").AssertNoContent()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertBadRequest(t *testing.T) {
	m := Method(GET, "").AssertBadRequest()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertUnauthorized(t *testing.T) {
	m := Method(GET, "").AssertUnauthorized()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertForbidden(t *testing.T) {
	m := Method(GET, "").AssertForbidden()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNotFound(t *testing.T) {
	m := Method(GET, "").AssertNotFound()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertConflict(t *testing.T) {
	m := Method(GET, "").AssertConflict()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertGone(t *testing.T) {
	m := Method(GET, "").AssertGone()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertUnprocessableEntity(t *testing.T) {
	m := Method(GET, "").AssertUnprocessableEntity()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertStatus(t *testing.T) {
	m := Method(GET, "").AssertStatus(404)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertFunc(t *testing.T) {
	m := Method(GET, "").AssertFunc(func(c Context) (unmet error, err error) {
		return nil, nil
	})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertEqual(t *testing.T) {
	m := Method(GET, "").AssertEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNotEqual(t *testing.T) {
	m := Method(GET, "").AssertNotEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertLessThan(t *testing.T) {
	m := Method(GET, "").AssertLessThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertLessThanOrEqual(t *testing.T) {
	m := Method(GET, "").AssertLessThanOrEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertGreaterThan(t *testing.T) {
	m := Method(GET, "").AssertGreaterThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertGreaterThanOrEqual(t *testing.T) {
	m := Method(GET, "").AssertGreaterThanOrEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNotLessThan(t *testing.T) {
	m := Method(GET, "").AssertNotLessThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNotGreaterThan(t *testing.T) {
	m := Method(GET, "").AssertNotGreaterThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertMatch(t *testing.T) {
	m := Method(GET, "").AssertMatch("", "")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertContains(t *testing.T) {
	m := Method(GET, "").AssertContains("", "")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertType(t *testing.T) {
	m := Method(GET, "").AssertType("", Type[string]())
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNil(t *testing.T) {
	m := Method(GET, "").AssertNil(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertNotNil(t *testing.T) {
	m := Method(GET, "").AssertNotNil(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_AssertLen(t *testing.T) {
	m := Method(GET, "").AssertLen("foo", 3)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireOK(t *testing.T) {
	m := Method(GET, "").RequireOK()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireCreated(t *testing.T) {
	m := Method(GET, "").RequireCreated()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireAccepted(t *testing.T) {
	m := Method(GET, "").RequireAccepted()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNoContent(t *testing.T) {
	m := Method(GET, "").RequireNoContent()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireBadRequest(t *testing.T) {
	m := Method(GET, "").RequireBadRequest()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireUnauthorized(t *testing.T) {
	m := Method(GET, "").RequireUnauthorized()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireForbidden(t *testing.T) {
	m := Method(GET, "").RequireForbidden()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNotFound(t *testing.T) {
	m := Method(GET, "").RequireNotFound()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireConflict(t *testing.T) {
	m := Method(GET, "").RequireConflict()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireGone(t *testing.T) {
	m := Method(GET, "").RequireGone()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireUnprocessableEntity(t *testing.T) {
	m := Method(GET, "").RequireUnprocessableEntity()
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireStatus(t *testing.T) {
	m := Method(GET, "").RequireStatus(404)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireFunc(t *testing.T) {
	m := Method(GET, "").RequireFunc(func(c Context) (unmet error, err error) {
		return nil, nil
	})
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireEqual(t *testing.T) {
	m := Method(GET, "").RequireEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNotEqual(t *testing.T) {
	m := Method(GET, "").RequireNotEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireLessThan(t *testing.T) {
	m := Method(GET, "").RequireLessThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireLessThanOrEqual(t *testing.T) {
	m := Method(GET, "").RequireLessThanOrEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireGreaterThan(t *testing.T) {
	m := Method(GET, "").RequireGreaterThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireGreaterThanOrEqual(t *testing.T) {
	m := Method(GET, "").RequireGreaterThanOrEqual(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNotLessThan(t *testing.T) {
	m := Method(GET, "").RequireNotLessThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNotGreaterThan(t *testing.T) {
	m := Method(GET, "").RequireNotGreaterThan(1, 2)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireMatch(t *testing.T) {
	m := Method(GET, "").RequireMatch("", "")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireContains(t *testing.T) {
	m := Method(GET, "").RequireContains("", "")
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireType(t *testing.T) {
	m := Method(GET, "").RequireType("foo", Type[string]())
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNil(t *testing.T) {
	m := Method(GET, "").RequireNil(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireNotNil(t *testing.T) {
	m := Method(GET, "").RequireNotNil(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestMethod_RequireLen(t *testing.T) {
	m := Method(GET, "").RequireLen("foo", 3)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestAssertMockServiceCalled(t *testing.T) {
	m := Method(GET, "").AssertMockServiceCalled("mock", "/foos", GET)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestRequireMockServiceCalled(t *testing.T) {
	m := Method(GET, "").RequireMockServiceCalled("mock", "/foos", GET)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestAssertHasProperties(t *testing.T) {
	m := Method(GET, "").AssertHasProperties(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestRequireHasProperties(t *testing.T) {
	m := Method(GET, "").RequireHasProperties(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestAssertOnlyHasProperties(t *testing.T) {
	m := Method(GET, "").AssertOnlyHasProperties(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.False(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}

func TestRequireOnlyHasProperties(t *testing.T) {
	m := Method(GET, "").RequireOnlyHasProperties(nil)
	raw, ok := m.(*method)
	require.True(t, ok)
	assert.Len(t, raw.expectations, 1)
	assert.True(t, raw.expectations[0].IsRequired())
	assert.Len(t, raw.postOps, 1)
	assert.True(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
}
