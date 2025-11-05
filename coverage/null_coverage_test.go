package coverage

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNullCoverage(t *testing.T) {
	cov := NewNullCoverage()
	assert.NoError(t, cov.LoadSpec(nil))
	assert.False(t, cov.HasFailures())
	cov.ReportMet(nil, nil, nil, nil)
	assert.False(t, cov.HasFailures())
	cov.ReportSkipped(nil, nil, nil, nil)
	assert.False(t, cov.HasFailures())
	cov.ReportTiming(nil, nil, nil, time.Second, nil)
	assert.False(t, cov.HasFailures())
	cov.ReportUnmet(nil, nil, nil, nil, nil)
	assert.True(t, cov.HasFailures())
	cov.ReportFailure(nil, nil, nil, nil)
	assert.True(t, cov.HasFailures())
}
