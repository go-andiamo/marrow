package marrow

import (
	"bufio"
	"bytes"
	goctx "context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-andiamo/columbus"
	"github.com/go-andiamo/gopt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
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
	case func() (any, error):
		av, err = vt()
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
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == ':' || c == '.' {
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
		if after, ok := strings.CutPrefix(name, "env:"); ok {
			if s, ok := os.LookupEnv(after); ok {
				b.WriteString(s)
			} else {
				return "", fmt.Errorf("unresolved env var: %q", after)
			}
		} else if v, ok := vars[Var(name)]; ok {
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

type DefaultVarValue struct {
	Var   Var
	Value any
}

func (v DefaultVarValue) ResolveValue(ctx Context) (av any, err error) {
	var ok bool
	if av, ok = ctx.Vars()[v.Var]; !ok {
		av = v.Value
		ctx.SetVar(v.Var, av)
	}
	return av, err
}

func (v DefaultVarValue) String() string {
	return fmt.Sprintf("DefaultVar(%s, %v)", string(v.Var), v.Value)
}

// DefaultVar resolves to the named var
//
// if the named var is not set, it resolves to the value provided (and the var is also set to that value)
func DefaultVar(name any, value any) DefaultVarValue {
	var varName Var
	switch nt := name.(type) {
	case Var:
		varName = nt
	case string:
		varName = Var(nt)
	default:
		varName = Var(fmt.Sprintf("%v", name))
	}
	return DefaultVarValue{
		Var:   varName,
		Value: value,
	}
}

// Env is the name of an environment variable
//
// when resolved, the value is the current environment variable of the name
type Env string

func (e Env) ResolveValue(ctx Context) (av any, err error) {
	return os.Getenv(string(e)), nil
}

func (e Env) String() string {
	return "Env(" + string(e) + ")"
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

type JsonifyValue struct {
	Value any
}

// Jsonify resolves to a value using the supplied value and attempts to coerce it to a JSON representation (i.e. map[string]any / []any)
//
// if the value does not coerce to JSON, the resolve errors
//
// the initial value can be: string, []byte, or any map/slice/struct
func Jsonify(v any) JsonifyValue {
	return JsonifyValue{
		Value: v,
	}
}

func (v JsonifyValue) ResolveValue(ctx Context) (av any, err error) {
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		switch rvt := rv.(type) {
		case string:
			var jv any
			if err = json.Unmarshal([]byte(rvt), &jv); err == nil {
				av = jv
			}
		case []byte:
			var jv any
			if err = json.Unmarshal(rvt, &jv); err == nil {
				av = jv
			}
		case map[string]any:
			av = rvt
		case []any:
			av = rvt
		default:
			if rv != nil {
				to := reflect.TypeOf(rv)
				if to.Kind() == reflect.Slice || to.Kind() == reflect.Map || to.Kind() == reflect.Struct {
					var data []byte
					if data, err = json.Marshal(rv); err == nil {
						var jv any
						if err = json.Unmarshal(data, &jv); err == nil {
							av = jv
						}
					}
				} else {
					err = fmt.Errorf("invalid type for json coerce: %T", rv)
				}
			}
		}
	}
	return av, err
}

type ApiLogsValue struct {
	Last int
}

// ApiLogs will resolve to the api logs (as []string) for the current api being tested
//
// it will error if the tests are not being run against an api container image
func ApiLogs(last int) ApiLogsValue {
	return ApiLogsValue{
		Last: last,
	}
}

func (v ApiLogsValue) ResolveValue(ctx Context) (av any, err error) {
	if api := ctx.GetApiImage(); api != nil && api.Container() != nil {
		var r io.ReadCloser
		if r, err = api.Container().Logs(goctx.Background()); err == nil {
			defer func() {
				_ = r.Close()
			}()
			lines := make([]string, 0)
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if err = scanner.Err(); err != nil {
				err = fmt.Errorf("failed to read api logs: %w", err)
			} else {
				if v.Last > 0 {
					if len(lines) > v.Last {
						lines = lines[len(lines)-v.Last:]
					}
				}
				av = lines
			}
		}
	} else {
		err = errors.New("no api image for logs")
	}
	return av, err
}

func (v ApiLogsValue) String() string {
	return fmt.Sprintf("ApiLogs(%d)", v.Last)
}

type LenValue struct {
	Value any
}

// Len resolves to the length of the value (or resolved value)
//
// if the value (or resolved value) is not a string, map or slice - this always resolves to -1
func Len(value any) LenValue {
	return LenValue{
		Value: value,
	}
}

func (v LenValue) ResolveValue(ctx Context) (av any, err error) {
	av = -1
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		switch rvt := rv.(type) {
		case string:
			av = len(rvt)
		case map[string]any:
			av = len(rvt)
		case []any:
			av = len(rvt)
		default:
			if rv != nil {
				to := reflect.ValueOf(rv)
				if to.Kind() == reflect.Slice || to.Kind() == reflect.Map {
					av = to.Len()
				}
			}
		}
	}
	return av, err
}

func (v LenValue) String() string {
	return fmt.Sprintf("Len(%v)", v.Value)
}

type FirstValue struct {
	Value any
}

// First resolves to the first item (element) of the value (or resolved value)
//
// if the value (or resolved value) is not a slice (or an empty slice) - this always resolves to nil
func First(value any) FirstValue {
	return FirstValue{
		Value: value,
	}
}

func (v FirstValue) ResolveValue(ctx Context) (av any, err error) {
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		switch rvt := rv.(type) {
		case []any:
			if len(rvt) > 0 {
				av = rvt[0]
			}
		default:
			if rv != nil {
				to := reflect.ValueOf(rv)
				if to.Kind() == reflect.Slice && to.Len() > 0 {
					av = to.Index(0).Interface()
				}
			}
		}
	}
	return av, err
}

func (v FirstValue) String() string {
	return fmt.Sprintf("First(%v)", v.Value)
}

type LastValue struct {
	Value any
}

// Last resolves to the last item (element) of the value (or resolved value)
//
// if the value (or resolved value) is not a slice (or an empty slice) - this always resolves to nil
func Last(value any) LastValue {
	return LastValue{
		Value: value,
	}
}

func (v LastValue) ResolveValue(ctx Context) (av any, err error) {
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		switch rvt := rv.(type) {
		case []any:
			if len(rvt) > 0 {
				av = rvt[len(rvt)-1]
			}
		default:
			if rv != nil {
				to := reflect.ValueOf(rv)
				if to.Kind() == reflect.Slice && to.Len() > 0 {
					av = to.Index(to.Len() - 1).Interface()
				}
			}
		}
	}
	return av, err
}

func (v LastValue) String() string {
	return fmt.Sprintf("Last(%v)", v.Value)
}

type NthValue struct {
	Value any
	Index int
}

// Nth resolves to the nth item (element) of the value (or resolved value)
//
// if the value (or resolved value) is not a slice, an empty slice or the index is out-of-bounds - this always resolves to nil
func Nth(value any, index int) NthValue {
	return NthValue{
		Value: value,
		Index: index,
	}
}

func (v NthValue) ResolveValue(ctx Context) (av any, err error) {
	var rv any
	if rv, err = ResolveValue(v.Value, ctx); err == nil {
		switch rvt := rv.(type) {
		case []any:
			if l := len(rvt); l > 0 {
				i := v.Index
				if i < 0 {
					i = l + i
				}
				if i >= 0 && i < l {
					av = rvt[i]
				}
			}
		default:
			if rv != nil {
				to := reflect.ValueOf(rv)
				if to.Kind() == reflect.Slice {
					if l := to.Len(); l > 0 {
						i := v.Index
						if i < 0 {
							i = l + i
						}
						if i >= 0 && i < l {
							av = to.Index(i).Interface()
						}
					}
				}
			}
		}
	}
	return av, err
}

func (v NthValue) String() string {
	return fmt.Sprintf("Nth(%v, %d)", v.Value, v.Index)
}

type AndValue struct {
	Values []any
}

// And is a resolvable value
//
// it resolves by boolean ANDing all the supplied values (or their resolved value)
//
// Notes:
//   - if any of the values is not a bool, it errors
//   - short-circuits on first false
//   - if a value is an Expectation, the boolean is deduced from whether the expectation was met
//   - nil values are ignored
func And(values ...any) AndValue {
	return AndValue{
		Values: values,
	}
}

func (v AndValue) ResolveValue(ctx Context) (av any, err error) {
	bv := false
	for _, value := range v.Values {
		if value == nil {
			continue
		}
		if exp, ok := value.(Expectation); ok {
			var unmet error
			if unmet, err = exp.Met(ctx); err == nil {
				bv = unmet == nil
			}
		} else if av, err = ResolveValue(value, ctx); err == nil {
			if b, ok := av.(bool); ok {
				bv = b
			} else {
				err = fmt.Errorf("and value expects boolean - got type %T", av)
			}
		}
		if !bv || err != nil {
			break
		}
	}
	av = bv
	return av, err
}

func (v AndValue) String() string {
	var b strings.Builder
	for _, value := range v.Values {
		if value != nil {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			if exp, ok := value.(Expectation); ok {
				b.WriteString(exp.Name())
			} else {
				b.WriteString(fmt.Sprintf("%v", value))
			}
		}
	}
	return "And(" + b.String() + ")"
}

type OrValue struct {
	Values []any
}

// Or is a resolvable value
//
// it resolves by boolean ORing all the supplied values (or their resolved value)
//
// Notes:
//   - if any of the values is not a bool, it errors
//   - short-circuits on first true
//   - if a value is an Expectation, the boolean is deduced from whether the expectation was met
//   - nil values are ignored
func Or(values ...any) OrValue {
	return OrValue{
		Values: values,
	}
}

func (v OrValue) ResolveValue(ctx Context) (av any, err error) {
	bv := false
	for _, value := range v.Values {
		if value == nil {
			continue
		}
		if exp, ok := value.(Expectation); ok {
			var unmet error
			if unmet, err = exp.Met(ctx); err == nil {
				bv = unmet == nil
			}
		} else if av, err = ResolveValue(value, ctx); err == nil {
			if b, ok := av.(bool); ok {
				bv = b
			} else {
				err = fmt.Errorf("or value expects boolean - got type %T", av)
			}
		}
		if bv || err != nil {
			break
		}
	}
	av = bv
	return av, err
}

func (v OrValue) String() string {
	var b strings.Builder
	for _, value := range v.Values {
		if value != nil {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			if exp, ok := value.(Expectation); ok {
				b.WriteString(exp.Name())
			} else {
				b.WriteString(fmt.Sprintf("%v", value))
			}
		}
	}
	return "Or(" + b.String() + ")"
}

type FormMultipart struct {
	Parts []MultipartPart
}

// Multipart creates a FormMultipart for use as Method_.RequestBody
//
// the parts can Field or FileField
func Multipart(parts ...MultipartPart) FormMultipart {
	return FormMultipart{Parts: parts}
}

func (m FormMultipart) buildBody(ctx Context) (data []byte, dataLen int, contentType string, err error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for _, part := range m.Parts {
		if err = part.writePart(ctx, w); err != nil {
			return
		}
	}
	data = buf.Bytes()
	dataLen = buf.Len()
	contentType = w.FormDataContentType()
	return
}

type MultipartPart interface {
	writePart(ctx Context, w *multipart.Writer) error
}

type MultipartField struct {
	Name  string
	Value any
}

// Field creates a MultipartPart for use in Multipart
//
// the value can be any type including a resolvable value
func Field(name string, value any) MultipartPart {
	return MultipartField{Name: name, Value: value}
}

func (f MultipartField) writePart(ctx Context, w *multipart.Writer) (err error) {
	var av any
	if av, err = ResolveValue(f.Value, ctx); err == nil {
		var data []byte
		switch avt := av.(type) {
		case []byte:
			data = avt
		case string:
			data = []byte(avt)
		default:
			data = []byte(fmt.Sprintf("%v", av))
		}
		var fw io.Writer
		if fw, err = w.CreateFormField(f.Name); err == nil {
			_, err = fw.Write(data)
		}
	}
	return err
}

type MultipartFile struct {
	FieldName   string
	FileName    string
	Source      any
	ContentType string
}

// FileField creates a MultipartPart for use in Multipart
//
// the source is the source data of the file - it can be a resolvable
// and the type must be either []byte or string.  A string value indicates a filename to load the source data from
func FileField(fieldName string, fileName string, source any, contentType string) MultipartPart {
	return MultipartFile{
		FieldName:   fieldName,
		FileName:    fileName,
		Source:      source,
		ContentType: contentType,
	}
}

func (f MultipartFile) writePart(ctx Context, w *multipart.Writer) (err error) {
	var src any
	if src, err = ResolveValue(f.Source, ctx); err == nil {
		var data []byte
		useFilename := f.FileName
		switch st := src.(type) {
		case []byte:
			data = st
		case string:
			if useFilename == "" {
				useFilename = filepath.Base(st)
			}
			data, err = os.ReadFile(st)
		default:
			err = fmt.Errorf("unsupported multipart file source type %T", src)
		}
		if err == nil {
			fieldName := f.FieldName
			if fieldName == "" {
				fieldName = "file"
			}
			header := make(textproto.MIMEHeader)
			disp := fmt.Sprintf(`form-data; name=%q`, fieldName)
			if useFilename != "" {
				disp += fmt.Sprintf(`; filename=%q`, useFilename)
			}
			header.Set("Content-Disposition", disp)
			contentType := f.ContentType
			if contentType == "" {
				contentType = http.DetectContentType(data)
			}
			header.Set("Content-Type", contentType)
			var fw io.Writer
			if fw, err = w.CreatePart(header); err == nil {
				_, err = fw.Write(data)
			}
		}
	}
	return err
}

type FormUrlEncoded struct {
	Values map[string]any
}

// UrlEncoded creates a FormUrlEncoded for use as Method_.RequestBody
//
// the request body is encoded as "application/x-www-form-urlencoded"
func UrlEncoded(values ...any) FormUrlEncoded {
	vm := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); {
		key := values[i]
		switch vt := key.(type) {
		case map[string]string:
			i++
			for k, v := range vt {
				vm[k] = v
			}
		case map[string]any:
			i++
			for k, v := range vt {
				vm[k] = v
			}
		default:
			if i+1 < len(values) {
				vm[fmt.Sprintf("%v", key)] = values[i+1]
			}
			i += 2
		}
	}
	return FormUrlEncoded{Values: vm}
}

func (m FormUrlEncoded) buildBody(ctx Context) ([]byte, error) {
	vals := url.Values{}
	for k, v := range m.Values {
		if av, err := ResolveValue(v, ctx); err == nil {
			switch avt := av.(type) {
			case []string:
				for _, s := range avt {
					vals.Add(k, s)
				}
			case []any:
				for _, item := range avt {
					vals.Add(k, fmt.Sprintf("%v", item))
				}
			default:
				vals.Set(k, fmt.Sprintf("%v", av))
			}
		} else {
			return nil, err
		}
	}
	return []byte(vals.Encode()), nil
}

type ApiCallValue struct {
	Method  MethodName
	Url     any
	Body    any
	Headers Headers
}

type Headers map[string]any

// ApiCall makes a call to the API under test - but not included as a test
//
// this is useful for preparatory calls before/after a test endpoint is called
//
// the value is resolved to the result of the call, represented as JSON map[string]any with the following properties...
//   - "status" the response status code
//   - "body" the response body (as []byte)
//   - "headers" a map[string]any of the response headers
func ApiCall(method MethodName, url any, body any, headers Headers) ApiCallValue {
	return ApiCallValue{
		Method:  method,
		Url:     url,
		Body:    body,
		Headers: headers,
	}
}

func (v ApiCallValue) ResolveValue(ctx Context) (av any, err error) {
	var ap any
	if ap, err = ResolveValue(v.Url, ctx); err == nil {
		var ab any
		if ab, err = ResolveValue(v.Body, ctx); err == nil {
			var ah any
			if ah, err = ResolveValue(map[string]any(v.Headers), ctx); err == nil {
				actualUrl := fmt.Sprintf("http://%s%v", ctx.Host(), ap)
				var body io.Reader
				switch abt := ab.(type) {
				case []byte:
					body = bytes.NewReader(abt)
				case string:
					body = strings.NewReader(abt)
				default:
					if ab != nil {
						to := reflect.TypeOf(ab)
						if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
							var data []byte
							if data, err = json.Marshal(ab); err == nil {
								body = bytes.NewReader(data)
							}
						} else {
							body = strings.NewReader(fmt.Sprintf("%v", ab))
						}
					}
				}
				if err == nil {
					var req *http.Request
					if req, err = http.NewRequestWithContext(ctx.Ctx(), string(v.Method), actualUrl, body); err == nil {
						if hm, ok := ah.(map[string]any); ok {
							for h, hv := range hm {
								req.Header.Set(h, fmt.Sprintf("%v", hv))
							}
						}
						var resp *http.Response
						if resp, err = ctx.DoRequest(req); err == nil {
							defer func() {
								_ = resp.Body.Close()
							}()
							var rBody []byte
							if rBody, err = io.ReadAll(resp.Body); err == nil {
								result := map[string]any{
									"status": resp.StatusCode,
									"body":   rBody,
								}
								hm := make(map[string]any, len(resp.Header))
								for k := range resp.Header {
									hm[k] = resp.Header.Get(k)
								}
								result["headers"] = hm
								av = result
							}
						}
					}
				}
			}
		}
	}
	return av, err
}

func (v ApiCallValue) String() string {
	return fmt.Sprintf("ApiCall(%s %s)", v.Method, v.Url)
}
