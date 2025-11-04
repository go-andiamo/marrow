package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-andiamo/columbus"
	"github.com/go-andiamo/gopt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ResolveValue takes any value and attempts to resolve it (using the supplied Context to obtain actual values - such as vars, response etc.)
//
// the value can be of any type - the following types are treated specially:
//   - anything that implements Resolvable (the value is deep resolved)
//   - BodyReader is executed to read the current context response body
//   - `func(any) (any, error)` is called with the current context response body
//   - TemplateString any var markers in the string are resolved
//   - `map[string]any`, `[]any`, JSON, JSONArray - all values within the map/slice are resolved
func ResolveValue(value any, ctx Context) (av any, err error) {
	av = value
	switch vt := value.(type) {
	case Resolvable:
		av, err = deepResolveValue(vt, ctx)
	case BodyReader:
		av, err = vt(ctx.CurrentBody())
	case func(any) (any, error):
		av, err = vt(ctx.CurrentBody())
	case TemplateString:
		av, err = resolveValueString(string(vt), ctx)
	case map[string]any:
		av, err = resolveMap(vt, ctx)
	case JSON:
		av, err = resolveMap(vt, ctx)
	case []any:
		av, err = resolveSlice(vt, ctx)
	case JSONArray:
		av, err = resolveSlice(vt, ctx)
	}
	return av, err
}

func deepResolveValue(v Resolvable, ctx Context) (av any, err error) {
	if av, err = v.ResolveValue(ctx); err == nil {
		if rv, ok := av.(Resolvable); ok {
			av, err = deepResolveValue(rv, ctx)
		}
	}
	return av, err
}

func resolveMap(m map[string]any, ctx Context) (any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if av, err := ResolveValue(v, ctx); err == nil {
			result[k] = av
		} else {
			return nil, err
		}
	}
	return result, nil
}

func resolveSlice(s []any, ctx Context) (any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		if av, err := ResolveValue(v, ctx); err == nil {
			result[i] = av
		} else {
			return nil, err
		}
	}
	return result, nil
}

func resolveValueString(s string, ctx Context) (string, error) {
	if !strings.Contains(s, "{$") {
		return s, nil
	}
	vars := ctx.Vars()
	var b strings.Builder
	unresolved := false
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], "{$")
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		j += i
		// count preceding backslashes...
		k := j - 1
		backslashes := 0
		for k >= i && s[k] == '\\' {
			backslashes++
			k--
		}
		// write the chunk before the backslashes...
		pre := j - backslashes
		b.WriteString(s[i:pre])
		if backslashes%2 == 1 {
			// escaped: keep one backslash as escape consumer, output "{$" literally
			// write the remaining backslashes (odd -> one fewer gets consumed)
			if backslashes > 1 {
				b.WriteString(s[pre : pre+backslashes-1])
			}
			b.WriteString("{$")
			i = j + 2
			continue
		}
		// unescaped placeholder: try to parse {$name}
		// scan name
		nameStart := j + 2
		nameEnd := nameStart
		for nameEnd < len(s) {
			c := s[nameEnd]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
				nameEnd++
				continue
			}
			break
		}
		if nameEnd == nameStart || nameEnd >= len(s) || s[nameEnd] != '}' {
			// malformed token; treat literally
			b.WriteString("{$")
			i = nameStart
			continue
		}
		name := s[nameStart:nameEnd]
		if v, ok := vars[Var(name)]; ok {
			if av, err := ResolveValue(v, ctx); err != nil {
				return "", err
			} else {
				b.WriteString(fmt.Sprintf("%v", av))
			}
		} else {
			// leave placeholder as-is and flag unresolved
			b.WriteString(s[j : nameEnd+1])
			unresolved = true
		}
		i = nameEnd + 1
	}
	if unresolved {
		return "", fmt.Errorf("unresolved variables in string %q", b.String())
	}
	return b.String(), nil
}

func stringifyValue(v any) string {
	vs := ""
	switch vt := v.(type) {
	case nil:
		vs = "<nil>"
	case fmt.Stringer:
		vs = vt.String()
	case string:
		vs = strconv.Quote(vt)
	default:
		vs = fmt.Sprintf("%v", vt)
	}
	return vs
}

const (
	FIRST = "FIRST" // FIRST is a special token for path in JsonPath - and means resolve to first item in slice
	LAST  = "LAST"  // LAST is a special token for path in JsonPath - and means resolve to last item in slice
	LEN   = "LEN"   // LEN is a special token for path in JsonPath - and means resolve to length of slice
)

func resolveJsonPath(v any, path string) (av any, err error) {
	if v == nil {
		return nil, fmt.Errorf("json path %q into nil", path)
	}
	switch vt := v.(type) {
	case map[string]any:
		if path == "" || path == "." {
			av = vt
		} else if o, _ := gopt.ExtractJsonPath[any](vt, path); o.IsPresent() {
			av = o.Default(nil)
		} else {
			return nil, fmt.Errorf("json path %q does not exist", path)
		}
	default:
		vo := reflect.ValueOf(v)
		if vo.Kind() == reflect.Slice {
			switch strings.ToUpper(path) {
			case "", ".":
				av = v
			case LEN:
				av = vo.Len()
			case FIRST, "0":
				if vo.Len() > 0 {
					av = vo.Index(0).Interface()
				} else {
					err = fmt.Errorf("json path %q into empty array", path)
				}
			case LAST:
				if l := vo.Len(); l > 0 {
					av = vo.Index(l - 1).Interface()
				} else {
					err = fmt.Errorf("json path %q into empty array", path)
				}
			default:
				if i, cerr := strconv.Atoi(path); cerr == nil {
					if l := vo.Len(); l > 0 {
						if i >= 0 && i < l {
							av = vo.Index(i).Interface()
						} else if i < 0 && (l+i) >= 0 {
							av = vo.Index(l + i).Interface()
						} else {
							err = fmt.Errorf("json path %q array index out of range", path)
						}
					} else {
						err = fmt.Errorf("json path %q into empty array", path)
					}
				} else {
					return nil, fmt.Errorf("json path %q invalid array index", path)
				}
			}
		} else {
			return nil, fmt.Errorf("json path %q into non object/array", path)
		}
	}
	return av, err
}

// Resolvable is an interface to be implemented by any type where the value is to be resolved
type Resolvable interface {
	ResolveValue(ctx Context) (av any, err error)
}

// BodyReader is a func that is called to resolve the value of the current context response body
//
// is specially treated by ResolveValue
//
// see also Body
type BodyReader func(body any) (any, error)

// Var is the name of a variable in the current Context
//
// when resolved, if the variable name is not set in the current Context it will cause an error in the test
type Var string

func (v Var) ResolveValue(ctx Context) (av any, err error) {
	vars := ctx.Vars()
	if vv, ok := vars[v]; ok {
		av = vv
	} else {
		err = fmt.Errorf("unknown variable %q", string(v))
	}
	return av, err
}

func (v Var) String() string {
	return "Var(" + string(v) + ")"
}

// StatusCode is a type that indicates resolve to the current Context response status code
type StatusCode int

func (StatusCode) ResolveValue(ctx Context) (av any, err error) {
	if response := ctx.CurrentResponse(); response == nil {
		return nil, errors.New("response is nil")
	} else {
		return response.StatusCode, nil
	}
}

// ResponseHeader is a type that will resolve to the specified header in the current Context response
//
// example:
//
//	ResponseHeader("Content-Type")
//
// will resolve to the "Content-Type" header vale in the current Context response
type ResponseHeader string

func (v ResponseHeader) ResolveValue(ctx Context) (av any, err error) {
	if response := ctx.CurrentResponse(); response == nil {
		return nil, errors.New("response is nil")
	} else {
		return response.Header.Get(string(v)), nil
	}
}

// ResponseCookie is a type that will resolve to the specified named cookie in the current Context response
//
// example:
//
//	ResponseCookie("session")
//
// will resolve to the named cookie "session" in the current Context response
//
// The resolved value is a `map[string]any` representation of a http.Cookie
type ResponseCookie string

func (v ResponseCookie) ResolveValue(ctx Context) (av any, err error) {
	if response := ctx.CurrentResponse(); response == nil {
		return nil, errors.New("response is nil")
	} else {
		for _, c := range response.Cookies() {
			if c.Name == string(v) {
				av = map[string]any{
					"Name":        c.Name,
					"Value":       c.Value,
					"Quoted":      c.Quoted,
					"Path":        c.Path,
					"Domain":      c.Domain,
					"Expires":     c.Expires.Format(time.RFC3339),
					"RawExpires":  c.RawExpires,
					"MaxAge":      c.MaxAge,
					"Secure":      c.Secure,
					"HttpOnly":    c.HttpOnly,
					"SameSite":    int(c.SameSite),
					"Partitioned": c.Partitioned,
					"Raw":         c.Raw,
					"Unparsed":    c.Unparsed,
				}
				break
			}
		}
	}
	return
}

type QueryValue struct {
	DbName string
	Query  string
	Args   []any
}

// Query resolves to a value obtained by executing the query and args against the named database
//
// Notes:
//   - if only one supporting database is used in tests, the dbName can be just ""
//   - the query **must** start with "SELECT "
//   - if there is only one selected column, the resolved value will be the value in that column
//   - if there are multiple columns selected, the resolved value will be a `map[string]any` representation of the row
func Query(dbName string, query string, args ...any) QueryValue {
	return QueryValue{
		DbName: dbName,
		Query:  query,
		Args:   args,
	}
}

func (v QueryValue) String() string {
	var b strings.Builder
	b.WriteString("Query(")
	b.WriteString(strconv.Quote(v.Query))
	for _, arg := range v.Args {
		b.WriteString(", ")
		b.WriteString(stringifyValue(arg))
	}
	b.WriteRune(')')
	return b.String()
}

func (v QueryValue) ResolveValue(ctx Context) (av any, err error) {
	var db *sql.DB
	if db = ctx.Db(v.DbName); db == nil {
		return nil, errors.New("db is nil")
	}
	query := v.Query
	if !strings.HasPrefix(strings.ToUpper(query), "SELECT ") {
		return nil, errors.New("query must start with \"SELECT\"")
	}
	if query, err = resolveValueString(query[6:], ctx); err == nil {
		args := make([]any, len(v.Args))
		for i, arg := range v.Args {
			var argV any
			if argV, err = ResolveValue(arg, ctx); err != nil {
				return
			} else {
				args[i] = argV
			}
		}
		var mapper columbus.Mapper
		if mapper, err = columbus.NewMapper(query, columbus.Query("")); err == nil {
			var row map[string]any
			if row, err = mapper.ExactlyOneRow(ctx.Ctx(), db, args); err == nil {
				if len(row) == 1 {
					for _, cv := range row {
						av = cv
						break
					}
				} else {
					av = row
				}
			}
		}
	}
	return av, err
}

type QueryRowsValue struct {
	DbName string
	Query  string
	Args   []any
}

// QueryRows resolves to a value obtained by executing the query and args against the named database
//
// Notes:
//   - if only one supporting database is used in tests, the dbName can be just ""
//   - the query **must** start with "SELECT "
//   - the resolved value is a `[]map[string]any` representation of the selected rows and columns
func QueryRows(dbName string, query string, args ...any) QueryRowsValue {
	return QueryRowsValue{
		DbName: dbName,
		Query:  query,
		Args:   args,
	}
}

func (v QueryRowsValue) ResolveValue(ctx Context) (av any, err error) {
	var db *sql.DB
	if db = ctx.Db(v.DbName); db == nil {
		return nil, errors.New("db is nil")
	}
	query := v.Query
	if !strings.HasPrefix(strings.ToUpper(query), "SELECT ") {
		return nil, errors.New("query must start with \"SELECT\"")
	}
	if query, err = resolveValueString(query[6:], ctx); err == nil {
		args := make([]any, len(v.Args))
		for i, arg := range v.Args {
			var argV any
			if argV, err = ResolveValue(arg, ctx); err != nil {
				return
			} else {
				args[i] = argV
			}
		}
		var mapper columbus.Mapper
		if mapper, err = columbus.NewMapper(query, columbus.Query("")); err == nil {
			var rows []map[string]any
			if rows, err = mapper.Rows(ctx.Ctx(), db, args); err == nil {
				av = rows
			}
		}
	}
	return av, err
}

// BodyPath resolves to a value using the supplied path against the current Context response body
//
// example:
//
//	BodyPath("foo.bar")
//
// will resolve to the value of property foo.bar in the current Context response body
type BodyPath string

func (v BodyPath) ResolveValue(ctx Context) (av any, err error) {
	if body := ctx.CurrentBody(); body == nil {
		err = errors.New("body is nil")
	} else {
		av, err = resolveJsonPath(body, string(v))
	}
	return av, err
}

type JsonPathValue struct {
	Value any
	Path  string
}

// JsonPath resolves to a value using the supplied path on the supplied value
//
// example:
//
//	JsonPath(Var("foo"), "bar")
//
// will resolve to the value of Var("foo") and return the value of property "bar" within that
func JsonPath(v any, path string) JsonPathValue {
	return JsonPathValue{
		Path:  path,
		Value: v,
	}
}

func (v JsonPathValue) String() string {
	return fmt.Sprintf("JsonPath(%s, %q)", stringifyValue(v.Value), v.Path)
}

func (v JsonPathValue) ResolveValue(ctx Context) (av any, err error) {
	if av, err = ResolveValue(v.Value, ctx); err == nil {
		av, err = resolveJsonPath(av, v.Path)
	}
	return av, err
}

type JsonTraverseValue struct {
	Value any
	Steps []any
}

// JsonTraverse resolves to a value using the supplied path steps on the supplied value
//
// example:
//
//	JsonTraverse(Var("foo"), FIRST, "bar", LAST)
//
// if the Var("foo") resolved to...
//
//	[
//	  {
//	    "bar": ["aaa", "bbb", "ccc"]
//	  }
//	]
//
// the final resolved value would be "ccc"
func JsonTraverse(v any, pathSteps ...any) JsonTraverseValue {
	return JsonTraverseValue{
		Value: v,
		Steps: pathSteps,
	}
}

func (v JsonTraverseValue) ResolveValue(ctx Context) (av any, err error) {
	if av, err = ResolveValue(v.Value, ctx); err == nil {
		var pathTo strings.Builder
		for _, step := range v.Steps {
			var seg string
			switch t := step.(type) {
			case string:
				seg = t
				if seg != "" && seg != "." {
					if pathTo.Len() > 0 {
						pathTo.WriteByte('.')
					}
					pathTo.WriteString(t)
				}
			case int:
				seg = strconv.Itoa(t)
				pathTo.WriteString("[" + seg + "]")
			default:
				seg = fmt.Sprintf("%v", t)
				if seg != "" && seg != "." {
					if pathTo.Len() > 0 {
						pathTo.WriteByte('.')
					}
					pathTo.WriteString(seg)
				}
			}
			if seg == "" || seg == "." {
				continue
			}
			if av, err = resolveJsonPath(av, seg); err != nil {
				err = fmt.Errorf("failed to traverse json path %q: %w", pathTo.String(), err)
				break
			}
		}
	}
	return av, err
}

// TemplateString is a string type that can contain variable markers - which are resolved to produce the final string
//
// variable markers are in the format "{$name}"
//
// when being resolved (against a Context), any variables not found will cause an error in the test
//
// to escape the marker start, use "\{$"
type TemplateString string

// JSON is shorthand for a `map[string]any` representation of a json object
//
// it is also treated specially by ResolveValue - where all values are deep resolved
type JSON map[string]any

// JSONArray is shorthand for a `[]any` representation of a json array
//
// it is also treated specially by ResolveValue - where all values are deep resolved
type JSONArray []any

type BodyValue string

const (
	// Body is shorthand for the response body
	Body BodyValue = "Body"
)

func (BodyValue) ResolveValue(ctx Context) (any, error) {
	if body := ctx.CurrentBody(); body != nil {
		return body, nil
	}
	return nil, nil
}

func (BodyValue) String() string {
	return "Body"
}
