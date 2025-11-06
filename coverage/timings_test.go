package coverage

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestTimings_Outliers(t *testing.T) {
	mk := func(ns ...int64) Timings {
		out := make(Timings, len(ns))
		for i, n := range ns {
			out[i] = Timing{Duration: time.Duration(n)}
		}
		return out
	}
	testCases := []struct {
		name       string
		in         Timings
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

func TestTimings_Stats(t *testing.T) {
	t.Run("empty false", func(t *testing.T) {
		timings := Timings{}
		_, ok := timings.Stats(false)
		assert.False(t, ok)
	})
	t.Run("all same (sample)", func(t *testing.T) {
		timings := Timings{
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
		}
		stats, ok := timings.Stats(true)
		assert.True(t, ok)
		assert.True(t, stats.Sample)
		assert.Equal(t, 4, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("sample 1", func(t *testing.T) {
		timings := Timings{
			{
				Duration: time.Second,
			},
		}
		stats, ok := timings.Stats(true)
		assert.True(t, ok)
		assert.True(t, stats.Sample)
		assert.Equal(t, 1, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("all same (population)", func(t *testing.T) {
		timings := Timings{
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second,
			},
		}
		stats, ok := timings.Stats(false)
		assert.True(t, ok)
		assert.False(t, stats.Sample)
		assert.Equal(t, 4, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("various", func(t *testing.T) {
		timings := Timings{
			{
				Duration: time.Second + (10 * time.Millisecond),
			},
			{
				Duration: time.Second + (20 * time.Millisecond),
			},
			{
				Duration: time.Second,
			},
			{
				Duration: time.Second + (30 * time.Millisecond),
			},
			{
				Duration: time.Second + (40 * time.Millisecond),
			},
		}
		stats, ok := timings.Stats(false)
		assert.True(t, ok)
		assert.False(t, stats.Sample)
		assert.Equal(t, 5, stats.Count)
		assert.Equal(t, time.Second+(20*time.Millisecond), stats.Mean)
		assert.Greater(t, stats.StdDev, 14*time.Millisecond)
		assert.Less(t, stats.StdDev, 15*time.Millisecond)
		assert.Less(t, stats.Variance, float64(0.001))
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, "1.04s", stats.Maximum.String())
		assert.Equal(t, "1.02s", stats.P50.String())
		assert.Equal(t, "1.036s", stats.P90.String())
		assert.Equal(t, "1.0396s", stats.P99.String())
	})
}

func TestTimings_StatsTTFB(t *testing.T) {
	t.Run("empty false", func(t *testing.T) {
		timings := Timings{}
		_, ok := timings.StatsTTFB(false)
		assert.False(t, ok)
	})
	t.Run("all same (sample)", func(t *testing.T) {
		timings := Timings{
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
		}
		stats, ok := timings.StatsTTFB(true)
		assert.True(t, ok)
		assert.True(t, stats.Sample)
		assert.Equal(t, 4, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("sample 1", func(t *testing.T) {
		timings := Timings{
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
		}
		stats, ok := timings.StatsTTFB(true)
		assert.True(t, ok)
		assert.True(t, stats.Sample)
		assert.Equal(t, 1, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("all same (population)", func(t *testing.T) {
		timings := Timings{
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
		}
		stats, ok := timings.StatsTTFB(false)
		assert.True(t, ok)
		assert.False(t, stats.Sample)
		assert.Equal(t, 4, stats.Count)
		assert.Equal(t, time.Second, stats.Mean)
		assert.Equal(t, time.Duration(0), stats.StdDev)
		assert.Equal(t, float64(0), stats.Variance)
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, time.Second, stats.Maximum)
		assert.Equal(t, time.Second, stats.P50)
		assert.Equal(t, time.Second, stats.P90)
		assert.Equal(t, time.Second, stats.P99)
	})
	t.Run("various", func(t *testing.T) {
		timings := Timings{
			{
				Trace: &TraceTiming{TTFB: time.Second + (10 * time.Millisecond)},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second + (20 * time.Millisecond)},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second + (30 * time.Millisecond)},
			},
			{
				Trace: &TraceTiming{TTFB: time.Second + (40 * time.Millisecond)},
			},
		}
		stats, ok := timings.StatsTTFB(false)
		assert.True(t, ok)
		assert.False(t, stats.Sample)
		assert.Equal(t, 5, stats.Count)
		assert.Equal(t, time.Second+(20*time.Millisecond), stats.Mean)
		assert.Greater(t, stats.StdDev, 14*time.Millisecond)
		assert.Less(t, stats.StdDev, 15*time.Millisecond)
		assert.Less(t, stats.Variance, float64(0.001))
		assert.Equal(t, time.Second, stats.Minimum)
		assert.Equal(t, "1.04s", stats.Maximum.String())
		assert.Equal(t, "1.02s", stats.P50.String())
		assert.Equal(t, "1.036s", stats.P90.String())
		assert.Equal(t, "1.0396s", stats.P99.String())
	})
	t.Run("missing trace", func(t *testing.T) {
		timings := Timings{
			{},
			{
				Trace: &TraceTiming{TTFB: time.Second + (10 * time.Millisecond)},
			},
		}
		_, ok := timings.StatsTTFB(false)
		assert.False(t, ok)
	})
}
