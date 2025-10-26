package coverage

import (
	"github.com/go-andiamo/splitter"
	"strings"
)

func normalizePath(path string) string {
	if parts, err := normSplitter.Split(path); err == nil {
		return "/" + strings.Join(parts, "/")
	} else {
		return path
	}
}

var normSplitter = splitter.MustCreateSplitter('/', splitter.CurlyBrackets).AddDefaultOptions(
	splitter.IgnoreEmptyFirst,
	splitter.IgnoreEmptyLast,
	&normCapture{})

type normCapture struct{}

func (nc *normCapture) Apply(s string, pos int, totalLen int, captured int, skipped int, isLast bool, subParts ...splitter.SubPart) (cap string, add bool, err error) {
	add = true
	cap = s
	if strings.HasPrefix(cap, "{") && strings.HasSuffix(cap, "}") {
		cap = "{}"
	}
	return
}
