package coverage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_normalizePath(t *testing.T) {
	nPath := normalizePath("/foo/{id}/bar/{id}")
	assert.Equal(t, "/foo/{}/bar/{}", nPath)

	nPath = normalizePath("/foo/{unbalanced")
	assert.Equal(t, "/foo/{unbalanced", nPath)
}
