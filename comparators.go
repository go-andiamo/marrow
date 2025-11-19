package marrow

import (
	"cmp"
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
)

//go:noinline
func newComparator(skip int, name string, v1, v2 any, c comp, not bool, required bool) Expectation {
	return &comparator{
		name:              name,
		v1:                v1,
		v2:                v2,
		comp:              c,
		not:               not,
		frame:             framing.NewFrame(skip),
		commonExpectation: commonExpectation{required: required},
	}
}

type comparator struct {
	name  string
	v1    any
	v2    any
	comp  comp
	not   bool
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*comparator)(nil)

//var _ Runnable = (*comparator)(nil)

func (c *comparator) Name() string {
	if c.name != "" {
		return c.name
	}
	return c.string()
}

func (c *comparator) newComparisonUnmet(msg string, ov1, ov2 OperandValue) UnmetError {
	result := &unmetError{
		msg:          msg,
		name:         c.Name(),
		isComparator: true,
		comparator:   c.string(),
		left:         ov1,
		right:        ov2,
		frame:        c.frame,
	}
	if result.msg == "" {
		result.msg = "expected " + c.string()
	}
	return result
}

func (c *comparator) Met(ctx Context) (unmet error, err error) {
	ov1 := OperandValue{
		Original: c.v1,
	}
	ov2 := OperandValue{
		Original: c.v2,
	}
	ok := false
	defer func() {
		if !ok && unmet == nil && err == nil {
			unmet = c.newComparisonUnmet("", ov1, ov2)
		}
	}()
	if ov1.Resolved, err = ResolveValue(ov1.Original, ctx); err != nil {
		err = fmt.Errorf("comparator failed to resolve value v1 (left): %w", err)
		return
	}
	if ov2.Resolved, err = ResolveValue(ov2.Original, ctx); err != nil {
		err = fmt.Errorf("comparator failed to resolve value v2 (right): %w", err)
		return
	}
	if ov1.Resolved == nil || ov2.Resolved == nil {
		switch {
		case c.comp == compEqual && ov1.Resolved == nil && ov2.Resolved == nil:
			ok = !c.not
		case c.comp == compEqual:
			ok = c.not
		default:
			unmet = c.newComparisonUnmet("cannot compare with nil", ov1, ov2)
		}
		return
	}
	compared := false
	comparison := 0
	switch vt1 := ov1.Resolved.(type) {
	case string:
		switch vt2 := ov2.Resolved.(type) {
		case string:
			compared = true
			comparison = strings.Compare(vt1, vt2)
		case bool:
			if c.comp == compEqual {
				compared = true
				comparison = strings.Compare(strings.ToLower(vt1), strconv.FormatBool(vt2))
			}
		case int:
			if ov1.Coerced, ov1.CoercionError = strconv.ParseInt(vt1, 10, 64); ov1.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(ov1.Coerced.(int64), int64(vt2))
			}
		case int64:
			if ov1.Coerced, ov1.CoercionError = strconv.ParseInt(vt1, 10, 64); ov1.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(ov1.Coerced.(int64), vt2)
			}
		case float64:
			if ov1.Coerced, ov1.CoercionError = strconv.ParseFloat(vt1, 64); ov1.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(ov1.Coerced.(float64), vt2)
			}
		case decimal.Decimal:
			if ov1.Coerced, ov1.CoercionError = decimal.NewFromString(vt1); ov1.CoercionError == nil {
				compared = true
				comparison = ov1.Coerced.(decimal.Decimal).Compare(vt2)
			}
		}
	case bool:
		if c.comp == compEqual {
			switch vt2 := ov2.Resolved.(type) {
			case bool:
				compared = true
				if vt1 != vt2 {
					comparison = -1
				}
			case string:
				compared = true
				ov1.Coerced = strconv.FormatBool(vt1)
				ov2.Coerced = strings.ToLower(vt2)
				comparison = strings.Compare(ov1.Coerced.(string), ov2.Coerced.(string))
			case int:
				compared = true
				if (vt1 && vt2 != 0) || (!vt1 && vt2 == 0) {
					comparison = 0
				} else {
					comparison = -1
				}
			case int64:
				compared = true
				if (vt1 && vt2 != 0) || (!vt1 && vt2 == 0) {
					comparison = 0
				} else {
					comparison = -1
				}
			case float64:
				compared = true
				if (vt1 && vt2 != 0) || (!vt1 && vt2 == 0) {
					comparison = 0
				} else {
					comparison = -1
				}
			case decimal.Decimal:
				compared = true
				if (vt1 && !vt2.IsZero()) || (!vt1 && vt2.IsZero()) {
					comparison = 0
				} else {
					comparison = -1
				}
			}
		}
	case int:
		switch vt2 := ov2.Resolved.(type) {
		case int:
			compared = true
			comparison = cmp.Compare(vt1, vt2)
		case int64:
			compared = true
			comparison = cmp.Compare(int64(vt1), vt2)
		case float64:
			compared = true
			comparison = cmp.Compare(float64(vt1), vt2)
		case decimal.Decimal:
			compared = true
			comparison = decimal.NewFromInt(int64(vt1)).Compare(vt2)
		case string:
			if ov2.Coerced, ov2.CoercionError = strconv.ParseInt(vt2, 10, 64); ov2.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(int64(vt1), ov2.Coerced.(int64))
			}
		case bool:
			if c.comp == compEqual {
				compared = true
				if (vt1 == 0 && !vt2) || (vt1 != 0 && vt2) {
					comparison = 0
				} else {
					comparison = -1
				}
			}
		}
	case int64:
		switch vt2 := ov2.Resolved.(type) {
		case int:
			compared = true
			comparison = cmp.Compare(vt1, int64(vt2))
		case int64:
			compared = true
			comparison = cmp.Compare(vt1, vt2)
		case float64:
			compared = true
			comparison = cmp.Compare(float64(vt1), vt2)
		case decimal.Decimal:
			compared = true
			comparison = decimal.NewFromInt(vt1).Compare(vt2)
		case string:
			if ov2.Coerced, ov2.CoercionError = strconv.ParseInt(vt2, 10, 64); ov2.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(vt1, ov2.Coerced.(int64))
			}
		case bool:
			if c.comp == compEqual {
				compared = true
				if (vt1 == 0 && !vt2) || (vt1 != 0 && vt2) {
					comparison = 0
				} else {
					comparison = -1
				}
			}
		}
	case float64:
		switch vt2 := ov2.Resolved.(type) {
		case float64:
			compared = true
			comparison = cmp.Compare(vt1, vt2)
		case decimal.Decimal:
			compared = true
			comparison = decimal.NewFromFloat(vt1).Compare(vt2)
		case int:
			compared = true
			comparison = cmp.Compare(vt1, float64(vt2))
		case int64:
			compared = true
			comparison = cmp.Compare(vt1, float64(vt2))
		case string:
			if ov2.Coerced, ov2.CoercionError = strconv.ParseFloat(vt2, 64); ov2.CoercionError == nil {
				compared = true
				comparison = cmp.Compare(vt1, ov2.Coerced.(float64))
			}
		case bool:
			if c.comp == compEqual {
				compared = true
				if (vt1 != 0 && vt2) || (vt1 == 0 && !vt2) {
					comparison = 0
				} else {
					comparison = -1
				}
			}
		}
	case decimal.Decimal:
		switch vt2 := ov2.Resolved.(type) {
		case decimal.Decimal:
			compared = true
			comparison = vt1.Compare(vt2)
		case float64:
			compared = true
			comparison = vt1.Compare(decimal.NewFromFloat(vt2))
		case int:
			compared = true
			comparison = vt1.Compare(decimal.NewFromInt(int64(vt2)))
		case int64:
			compared = true
			comparison = vt1.Compare(decimal.NewFromInt(vt2))
		case string:
			if ov2.Coerced, ov2.CoercionError = decimal.NewFromString(vt2); ov2.CoercionError == nil {
				compared = true
				comparison = vt1.Compare(ov2.Coerced.(decimal.Decimal))
			}
		case bool:
			if c.comp == compEqual {
				compared = true
				if (!vt1.IsZero() && vt2) || (vt1.IsZero() && !vt2) {
					comparison = 0
				} else {
					comparison = -1
				}
			}
		}
	}
	if !compared {
		unmet = c.newComparisonUnmet(fmt.Sprintf("cannot compare %s on: v1 (left) = %T, v2 (right) = %T", c.string(), ov1.Resolved, ov2.Resolved), ov1, ov2)
		return
	}
	switch c.comp {
	case compEqual:
		ok = comparison == 0
	case compLessThan:
		ok = comparison < 0
	case compGreaterThan:
		ok = comparison > 0
	case compLessOrEqualThan:
		ok = comparison <= 0
	case compGreaterOrEqualThan:
		ok = comparison >= 0
	}
	if c.not {
		ok = !ok
	}
	return
}

func (c *comparator) Frame() *framing.Frame {
	return c.frame
}

func (c *comparator) string() string {
	if c.not {
		return "NOT(" + compStrings[c.comp] + ")"
	}
	return compStrings[c.comp]
}

// ExpectEqual asserts that the supplied values are equal
//
// values can be any of:
//   - primitive type of string, bool, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectEqual(v1, v2 any) Expectation {
	return newComparator(1, "ExpectEqual", v1, v2, compEqual, false, false)
}

// ExpectNotEqual asserts that the supplied values are not equal
//
// values can be any of:
//   - primitive type of string, bool, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectNotEqual(v1, v2 any) Expectation {
	return newComparator(1, "ExpectNotEqual", v1, v2, compEqual, true, false)
}

// ExpectLessThan asserts that v1 is less than v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectLessThan(v1, v2 any) Expectation {
	return newComparator(1, "ExpectLessThan", v1, v2, compLessThan, false, false)
}

// ExpectLessThanOrEqual asserts that v1 is less than or equal to v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectLessThanOrEqual(v1, v2 any) Expectation {
	return newComparator(1, "ExpectLessThanOrEqual", v1, v2, compLessOrEqualThan, false, false)
}

// ExpectGreaterThan asserts that v1 is greater than v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectGreaterThan(v1, v2 any) Expectation {
	return newComparator(1, "ExpectGreaterThan", v1, v2, compGreaterThan, false, false)
}

// ExpectGreaterThanOrEqual asserts that v1 is greater than or equal to v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectGreaterThanOrEqual(v1, v2 any) Expectation {
	return newComparator(1, "ExpectGreaterThanOrEqual", v1, v2, compGreaterOrEqualThan, false, false)
}

// ExpectNotLessThan asserts that v1 is not less than v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectNotLessThan(v1, v2 any) Expectation {
	return newComparator(1, "ExpectNotLessThan", v1, v2, compLessThan, true, false)
}

// ExpectNotGreaterThan asserts that v1 is not greater than v2
//
// values can be any of:
//   - primitive type of string, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectNotGreaterThan(v1, v2 any) Expectation {
	return newComparator(1, "ExpectNotGreaterThan", v1, v2, compGreaterThan, true, false)
}

type comp int

const (
	compEqual comp = iota
	compLessThan
	compGreaterThan
	compLessOrEqualThan
	compGreaterOrEqualThan
)

var compStrings = map[comp]string{
	compEqual:              "==",
	compLessThan:           "<",
	compGreaterThan:        ">",
	compLessOrEqualThan:    "<=",
	compGreaterOrEqualThan: ">=",
}
