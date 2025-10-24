package marrow

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func Test_normalizePath(t *testing.T) {
	path := "/foo/{id}/bar/{id}"
	nPath := normalizePath(path)
	assert.Equal(t, "/foo/{}/bar/{}", nPath)
}

func TestCoverageTimings_Outliers(t *testing.T) {
	mk := func(ns ...int64) CoverageTimings {
		out := make(CoverageTimings, len(ns))
		for i, n := range ns {
			out[i] = CoverageTiming{Duration: time.Duration(n)}
		}
		return out
	}
	testCases := []struct {
		name       string
		in         CoverageTimings
		percentile float64
		expect     []int64
	}{
		{"empty", nil, 0.95, nil},
		{"p<=0 returns all sorted", mk(5, 1, 3), 0, []int64{1, 3, 5}},
		{"p>1 clamps to 1 (max block)", mk(7, 2, 5, 5, 5, 1), 2, []int64{7}},
		{"exact quantile tie", mk(3, 3, 4, 1, 2), 0.5, []int64{3, 3, 4}},
		{"interpolated threshold", mk(50, 20, 30, 40, 10), 0.9, []int64{50}},
		{"all equal", mk(5, 5, 5), 0.99, []int64{5, 5, 5}},
		{"single sample any p", mk(123), 0.42, []int64{123}},
		{"p==1 (max block ties)", mk(1, 9, 9, 3), 1.0, []int64{9, 9}},
		{"p NaN returns all sorted", mk(3, 1, 2), math.NaN(), []int64{1, 2, 3}},
		{"p Inf returns all sorted", mk(3, 1, 2), math.Inf(0), []int64{1, 2, 3}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.in.Outliers(tc.percentile)
			if tc.expect == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				got := make([]int64, len(result))
				for i, v := range result {
					got[i] = int64(v.Duration)
				}
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}
