package marrow

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
)

type queryParams map[string][]any

func (qp queryParams) add(name string, value any) {
	qp[name] = append(qp[name], value)
}

func (qp queryParams) encode(ctx Context) (string, error) {
	var buf strings.Builder
	for _, k := range slices.Sorted(maps.Keys(qp)) {
		vs := qp[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			switch v.(type) {
			case nil:
				buf.WriteString(keyEscaped)
			default:
				if av, err := ResolveValue(v, ctx); err == nil {
					buf.WriteString(keyEscaped)
					buf.WriteByte('=')
					buf.WriteString(url.QueryEscape(fmt.Sprintf("%v", av)))
				} else {
					return "", err
				}
				/*
					case Var:
						buf.WriteString(keyEscaped)
						buf.WriteByte('=')
						buf.WriteString(url.QueryEscape(fmt.Sprintf("%v", vars[string(value)])))
					default:
						buf.WriteString(keyEscaped)
						buf.WriteByte('=')
						buf.WriteString(url.QueryEscape(fmt.Sprintf("%v", v)))
				*/
			}
		}
	}
	if buf.Len() == 0 {
		return "", nil
	}
	return "?" + buf.String(), nil
}
