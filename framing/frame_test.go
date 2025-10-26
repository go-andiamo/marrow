package framing

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFrame(t *testing.T) {
	s := newFrameTest()
	f := s.frame
	assert.NotNil(t, f)
	assert.Equal(t, "TestNewFrame", f.Name)
	assert.Equal(t, 9, f.Line)
}

type frameTest struct {
	frame *Frame
}

//go:noinline
func newFrameTest() *frameTest {
	return &frameTest{
		frame: NewFrame(0),
	}
}
