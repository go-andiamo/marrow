package marrow

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_newComparator(t *testing.T) {
	c := newComparator(0, "", nil, nil, compEqual, false, false)
	assert.Equal(t, "<nil> == <nil>", c.Name())
	f := c.Frame()
	assert.NotNil(t, f)
	assert.Equal(t, t.Name(), f.Name)

	c = newComparator(0, "", nil, nil, compEqual, true, false)
	assert.Equal(t, "NOT(<nil> == <nil>)", c.Name())

	c = newComparator(0, "Test", nil, nil, compEqual, true, false)
	assert.Equal(t, "Test", c.Name())
}

func Test_comparator_Met_WithNils(t *testing.T) {
	t.Run("== both nil", func(t *testing.T) {
		c := newComparator(0, "", nil, nil, compEqual, false, false)
		unmet, err := c.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("NOT(==) both nil", func(t *testing.T) {
		c := newComparator(0, "", nil, nil, compEqual, true, false)
		unmet, err := c.Met(nil)
		assert.Error(t, unmet)
		assert.Equal(t, "expected NOT(<nil> == <nil>)", unmet.Error())
		assert.NoError(t, err)
	})
	t.Run("== one nil", func(t *testing.T) {
		c := newComparator(0, "", "foo", nil, compEqual, false, false)
		unmet, err := c.Met(nil)
		assert.Error(t, unmet)
		assert.Equal(t, "expected \"foo\" == <nil>", unmet.Error())
		assert.NoError(t, err)
	})
	t.Run("== one nil (flipped)", func(t *testing.T) {
		c := newComparator(0, "", nil, "foo", compEqual, false, false)
		unmet, err := c.Met(nil)
		assert.Error(t, unmet)
		assert.Equal(t, "expected <nil> == \"foo\"", unmet.Error())
		assert.NoError(t, err)
	})
	t.Run("> both nil", func(t *testing.T) {
		c := newComparator(0, "", nil, nil, compGreaterThan, false, false)
		unmet, err := c.Met(nil)
		assert.Error(t, unmet)
		assert.Equal(t, "cannot compare with nil", unmet.Error())
		assert.NoError(t, err)
	})
}

func Test_comparator_Met_WithResolveFailures(t *testing.T) {
	t.Run("left side", func(t *testing.T) {
		c := newComparator(0, "", Var("missing"), "foo", compEqual, false, false)
		unmet, err := c.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.Error(t, err)
		assert.Equal(t, "comparator failed to resolve value v1 (left): unknown variable \"missing\"", err.Error())
	})
	t.Run("right side", func(t *testing.T) {
		c := newComparator(0, "", "foo", Var("missing"), compEqual, false, false)
		unmet, err := c.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.Error(t, err)
		assert.Equal(t, "comparator failed to resolve value v2 (right): unknown variable \"missing\"", err.Error())
	})
}

func Test_comparator_Met_Values(t *testing.T) {
	testCases := []struct {
		v1          any
		v2          any
		comp        comp
		not         bool
		expectOk    bool
		expectErr   string
		expectV1Err bool
		expectV2Err bool
	}{
		{
			v1:       "foo",
			v2:       "foo",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       "foo",
			v2:       []byte("foo"),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       []byte("foo"),
			v2:       "foo",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       []byte("foo"),
			v2:       []byte("foo"),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        "foo",
			v2:        "foo",
			comp:      compEqual,
			not:       true,
			expectOk:  false,
			expectErr: "expected NOT(\"foo\" == \"foo\")",
		},
		{
			v1:       "true",
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       true,
			v2:       "true",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        "true",
			v2:        true,
			comp:      compGreaterThan,
			expectOk:  false,
			expectErr: "cannot compare \"true\" > true on: v1 (left) = string, v2 (right) = bool",
		},
		{
			v1:        true,
			v2:        "true",
			comp:      compGreaterThan,
			expectOk:  false,
			expectErr: "cannot compare true > \"true\" on: v1 (left) = bool, v2 (right) = string",
		},
		{
			v1:       "1",
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       "2",
			v2:       1,
			comp:     compGreaterThan,
			expectOk: true,
		},
		{
			v1:       "1",
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       "2",
			v2:       int64(1),
			comp:     compGreaterThan,
			expectOk: true,
		},
		{
			v1:       "1.101",
			v2:       1.101,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       "1.101",
			v2:       decimal.NewFromFloat(1.101),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       true,
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       true,
			v2:       false,
			comp:     compEqual,
			not:      true,
			expectOk: true,
		},
		{
			v1:       true,
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        true,
			v2:        0,
			comp:      compEqual,
			expectErr: "expected true == 0",
		},
		{
			v1:       1,
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        0,
			v2:        true,
			comp:      compEqual,
			expectErr: "expected 0 == true",
		},
		{
			v1:       true,
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        true,
			v2:        int64(0),
			comp:      compEqual,
			expectErr: "expected true == 0",
		},
		{
			v1:       int64(1),
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        int64(0),
			v2:        true,
			comp:      compEqual,
			expectErr: "expected 0 == true",
		},
		{
			v1:       true,
			v2:       1.0,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        true,
			v2:        0.0,
			comp:      compEqual,
			expectErr: "expected true == 0",
		},
		{
			v1:       1.0,
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        0.0,
			v2:        true,
			comp:      compEqual,
			expectErr: "expected 0 == true",
		},
		{
			v1:       true,
			v2:       decimal.NewFromFloat(1.0),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        true,
			v2:        decimal.NewFromFloat(0.0),
			comp:      compEqual,
			expectErr: "expected true == 0",
		},
		{
			v1:       decimal.NewFromFloat(1.0),
			v2:       true,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:        decimal.NewFromFloat(0.0),
			v2:        true,
			comp:      compEqual,
			expectErr: "expected 0 == true",
		},
		{
			v1:       1,
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       1,
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       1,
			v2:       float64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       1,
			v2:       "1",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       1,
			v2:       decimal.NewFromInt(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       int64(1),
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       int64(1),
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       int64(1),
			v2:       float64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       int64(1),
			v2:       "1",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       int64(1),
			v2:       decimal.NewFromInt(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       float64(1),
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       float64(1),
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       float64(1),
			v2:       float64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       float64(1),
			v2:       "1",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       float64(1),
			v2:       decimal.NewFromInt(1),
			comp:     compEqual,
			expectOk: true,
		},

		{
			v1:       decimal.NewFromInt(1),
			v2:       1,
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       decimal.NewFromInt(1),
			v2:       int64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       decimal.NewFromInt(1),
			v2:       float64(1),
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       decimal.NewFromInt(1),
			v2:       "1",
			comp:     compEqual,
			expectOk: true,
		},
		{
			v1:       decimal.NewFromInt(1),
			v2:       decimal.NewFromInt(1),
			comp:     compEqual,
			expectOk: true,
		},
		// various comparisons...
		{
			v1:       1,
			v2:       2,
			comp:     compLessThan,
			expectOk: true,
		},
		{
			v1:       2,
			v2:       2,
			comp:     compLessOrEqualThan,
			expectOk: true,
		},
		{
			v1:       2,
			v2:       1,
			comp:     compGreaterThan,
			expectOk: true,
		},
		{
			v1:       2,
			v2:       2,
			comp:     compGreaterOrEqualThan,
			expectOk: true,
		},
		// coercion errors...
		{
			v1:          "not a number",
			v2:          1,
			comp:        compEqual,
			expectErr:   "cannot compare \"not a number\" == 1 on: v1 (left) = string, v2 (right) = int",
			expectV1Err: true,
		},
		{
			v1:          "not a number",
			v2:          int64(1),
			comp:        compEqual,
			expectErr:   "cannot compare \"not a number\" == 1 on: v1 (left) = string, v2 (right) = int64",
			expectV1Err: true,
		},
		{
			v1:          "not a number",
			v2:          float64(1),
			comp:        compEqual,
			expectErr:   "cannot compare \"not a number\" == 1 on: v1 (left) = string, v2 (right) = float64",
			expectV1Err: true,
		},
		{
			v1:          "not a number",
			v2:          decimal.NewFromInt(1),
			comp:        compEqual,
			expectErr:   "cannot compare \"not a number\" == 1 on: v1 (left) = string, v2 (right) = decimal.Decimal",
			expectV1Err: true,
		},
		{
			v1:          1,
			v2:          "not a number",
			comp:        compEqual,
			expectErr:   "cannot compare 1 == \"not a number\" on: v1 (left) = int, v2 (right) = string",
			expectV2Err: true,
		},
		{
			v1:          int64(1),
			v2:          "not a number",
			comp:        compEqual,
			expectErr:   "cannot compare 1 == \"not a number\" on: v1 (left) = int64, v2 (right) = string",
			expectV2Err: true,
		},
		{
			v1:          float64(1),
			v2:          "not a number",
			comp:        compEqual,
			expectErr:   "cannot compare 1 == \"not a number\" on: v1 (left) = float64, v2 (right) = string",
			expectV2Err: true,
		},
		{
			v1:          decimal.NewFromInt(1),
			v2:          "not a number",
			comp:        compEqual,
			expectErr:   "cannot compare 1 == \"not a number\" on: v1 (left) = decimal.Decimal, v2 (right) = string",
			expectV2Err: true,
		},
		{
			v1:          decimal.NewFromInt(1),
			v2:          "not a number",
			comp:        compEqual,
			not:         true,
			expectErr:   "cannot compare NOT(1 == \"not a number\") on: v1 (left) = decimal.Decimal, v2 (right) = string",
			expectV2Err: true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			c := newComparator(0, "", tc.v1, tc.v2, tc.comp, tc.not, false)
			unmet, err := c.Met(nil)
			require.NoError(t, err)
			if tc.expectOk {
				assert.NoError(t, unmet)
				assert.NoError(t, err)
			} else {
				require.Error(t, unmet)
				assert.Equal(t, tc.expectErr, unmet.Error())
				umerr, is := unmet.(UnmetError)
				require.True(t, is)
				require.Error(t, umerr)
				if tc.expectV1Err {
					assert.Error(t, umerr.Left().CoercionError)
				} else {
					assert.NoError(t, umerr.Left().CoercionError)
				}
				if tc.expectV2Err {
					assert.Error(t, umerr.Right().CoercionError)
				} else {
					assert.NoError(t, umerr.Right().CoercionError)
				}
			}
		})
	}
}

func TestComparatorFunctions(t *testing.T) {
	testCases := []struct {
		value      Expectation
		expectName string
		expectComp comp
		expectNot  bool
	}{
		{
			value:      ExpectEqual(0, 0),
			expectName: "ExpectEqual",
			expectComp: compEqual,
			expectNot:  false,
		},
		{
			value:      ExpectNotEqual(0, 0),
			expectName: "ExpectNotEqual",
			expectComp: compEqual,
			expectNot:  true,
		},
		{
			value:      ExpectLessThan(0, 0),
			expectName: "ExpectLessThan",
			expectComp: compLessThan,
			expectNot:  false,
		},
		{
			value:      ExpectLessThanOrEqual(0, 0),
			expectName: "ExpectLessThanOrEqual",
			expectComp: compLessOrEqualThan,
			expectNot:  false,
		},
		{
			value:      ExpectGreaterThan(0, 0),
			expectName: "ExpectGreaterThan",
			expectComp: compGreaterThan,
			expectNot:  false,
		},
		{
			value:      ExpectGreaterThanOrEqual(0, 0),
			expectName: "ExpectGreaterThanOrEqual",
			expectComp: compGreaterOrEqualThan,
			expectNot:  false,
		},
		{
			value:      ExpectNotLessThan(0, 0),
			expectName: "ExpectNotLessThan",
			expectComp: compLessThan,
			expectNot:  true,
		},
		{
			value:      ExpectNotGreaterThan(0, 0),
			expectName: "ExpectNotGreaterThan",
			expectComp: compGreaterThan,
			expectNot:  true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			assert.Equal(t, tc.expectName, tc.value.Name())
			assert.NotNil(t, tc.value.Frame())
			assert.False(t, tc.value.IsRequired())
			assert.Equal(t, tc.expectComp, tc.value.(*comparator).comp)
			assert.Equal(t, tc.expectNot, tc.value.(*comparator).not)
		})
	}
}
