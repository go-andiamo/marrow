package framing

import (
	"runtime"
	"strings"
)

// Framed is an interface implemented by many components with marrow
type Framed interface {
	Frame() *Frame
}

//go:noinline
func NewFrame(skip int) (f *Frame) {
	stack := make([]uintptr, 1)
	if l := runtime.Callers(3+skip, stack[:]); l > 0 {
		f = &Frame{
			Pc: stack[0],
		}
		f.fill()
	}
	return f
}

// Frame is information about where something was declared
type Frame struct {
	File    string
	Line    int
	Name    string
	Package string
	Pc      uintptr
}

func (f *Frame) fill() {
	fn := runtime.FuncForPC(f.Pc)
	f.File, f.Line = fn.FileLine(f.Pc - 1)
	name := fn.Name()
	pkg := ""
	if last := strings.LastIndex(name, "/"); last >= 0 {
		pkg += name[:last] + "/"
		name = name[last+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}
	name = strings.Replace(name, "·", ".", -1)
	f.Name, f.Package = name, pkg
}
