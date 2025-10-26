package coverage

import (
	"github.com/go-andiamo/marrow/common"
	"math"
	"net/http"
	"sort"
	"time"
)

type Timing struct {
	Endpoint common.Endpoint
	Method   common.Method
	Request  *http.Request
	Duration time.Duration
}

type Timings []Timing

type TimingStats struct {
	// Mean is the mean average response time. Gives a sense of typical latency under the current test conditions
	Mean time.Duration
	// StdDev is the standard deviation - how much individual response times deviate from the mean.
	//
	// A small StdDev means responses are consistent; a large one means erratic latency
	StdDev time.Duration // sqrt(Variance)
	// Variance is how much response times vary - in squared seconds.
	//
	// The lower the value, the better.  A high value indicates jitter and inconsistent response times
	Variance float64 // in seconds²
	// Minimum is the fastest observed response time. Indicates best-case performance
	Minimum time.Duration
	// Maximum is the slowest observed response time. Often reveals worst-case outliers (e.g., cold starts, timeouts)
	Maximum time.Duration
	// P50 is the median response time - half the runs were faster, half slower.
	//
	// Unlike Mean, it’s robust to outliers.  Indicates “Typical” latency.
	P50 time.Duration
	// P90 is the 90th percentile of response times - 9 out of 10 runs were this fast or faster.
	//
	// Useful for SLO/SLA checks.
	P90 time.Duration
	// P99 is the 99th percentile of response times - highlights rare but serious tail latencies.
	P99 time.Duration
	// Count is how many timings the stats are based on.
	//
	// Important for trustworthiness: low counts mean less reliable percentiles.
	Count int
	// Sample is whether the Variance / StdDev were computed using sample (n-1) or population (n) denominator — mostly for internal correctness.
	Sample bool
}

const sec = float64(time.Second)

// Stats creates a TimingStats from the Timings
//
// sample arg determines whether resulting TimingStats.StdDev & TimingStats.Variance are computed
// using sample (n-1) or population (n)
func (ct Timings) Stats(sample bool) (TimingStats, bool) {
	if len(ct) == 0 {
		return TimingStats{}, false
	}
	n := 0.0
	meanSec := 0.0
	m2Sec2 := 0.0
	minD := ct[0].Duration
	maxD := ct[0].Duration
	durations := make([]time.Duration, len(ct))
	for i, d := range ct {
		durations[i] = d.Duration
	}
	for _, d := range durations {
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
		x := float64(d) / sec
		n++
		delta := x - meanSec
		meanSec += delta / n
		m2Sec2 += delta * (x - meanSec)
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	percentile := func(p float64) time.Duration {
		pos := p * float64(len(durations)-1)
		i := int(math.Floor(pos))
		j := i + 1
		if j >= len(durations) {
			return durations[i]
		}
		f := pos - float64(i)
		// linear interpolation in nanoseconds (int64) with rounding
		a := float64(durations[i])
		b := float64(durations[j])
		return time.Duration(math.Round(a + f*(b-a)))
	}
	result := TimingStats{
		Sample:  sample,
		Mean:    time.Duration(math.Round(meanSec * sec)),
		Count:   len(ct),
		Maximum: maxD,
		Minimum: minD,
		P50:     percentile(0.5),
		P90:     percentile(0.9),
		P99:     percentile(0.99),
	}
	if sample {
		if n < 2 {
			return result, true
		}
		result.Variance = m2Sec2 / (n - 1.0)
	} else {
		result.Variance = m2Sec2 / n
	}
	result.StdDev = time.Duration(math.Round(math.Sqrt(result.Variance) * sec))
	return result, true
}

// Outliers returns the subset of timings whose Duration is at or above
// the given percentile threshold (upper-tail). For example, p=0.99 returns
// the slowest ~1% of requests. If timings is empty, returns nil.
// Percentile is clamped into [0,1]. p==0 returns all, p==1 returns only the max(es).
//
// Returned timings are sorted - slowest appearing last
func (ct Timings) Outliers(percentile float64) []Timing {
	if len(ct) == 0 {
		return nil
	}
	durs := make([]Timing, len(ct))
	copy(durs, ct)
	sort.Slice(durs, func(i, j int) bool {
		return durs[i].Duration < durs[j].Duration
	})
	// clamp percentile to max 1, or all for <= 0
	if math.IsNaN(percentile) || math.IsInf(percentile, 0) || percentile <= 0 {
		return durs
	} else if percentile > 1 {
		percentile = 1
	}
	pos := percentile * float64(len(durs)-1)
	i := int(math.Floor(pos))
	j := i + 1
	var start int
	var threshold time.Duration
	if j >= len(durs) {
		threshold = durs[i].Duration
		start = len(durs) - 1
	} else {
		f := pos - float64(i)
		a := float64(durs[i].Duration)
		b := float64(durs[j].Duration)
		threshold = time.Duration(math.Round(a + f*(b-a)))
		start = j
	}
	for bk := start - 1; bk >= 0 && durs[bk].Duration >= threshold; bk-- {
		start--
	}
	return durs[start:]
}
