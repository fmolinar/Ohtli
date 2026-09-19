// Package gnmi wraps the raw gNMI gRPC client (Capabilities, Get,
// Subscribe) used by the read-only controller (README.md section 7,
// phase 3). It intentionally avoids ygnmi/ygot codegen from YANG models:
// for a read-only MVP against whatever OpenConfig paths a target's
// Capabilities response actually advertises, a small hand-rolled path
// parser is enough, and it keeps the controller buildable without a YANG
// toolchain.
package gnmi

import (
	"fmt"
	"strconv"
	"strings"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// ParsePath parses a gNMI path string such as
// "/interfaces/interface[name=eth1]/state/oper-status" into a gnmipb.Path.
// It supports at most one key predicate per path element, which covers
// every OpenConfig path used in this controller.
func ParsePath(path string) (*gnmipb.Path, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &gnmipb.Path{}, nil
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return &gnmipb.Path{}, nil
	}

	var elems []*gnmipb.PathElem
	for _, segment := range splitPathSegments(path) {
		elem, err := parseSegment(segment)
		if err != nil {
			return nil, fmt.Errorf("gnmi: parsing path %q: %w", path, err)
		}
		elems = append(elems, elem)
	}
	return &gnmipb.Path{Elem: elems}, nil
}

// splitPathSegments splits on '/' while treating "[...]" as opaque, so a
// '/' can't appear inside a key value in these paths (it never does for
// the OpenConfig paths this controller uses).
func splitPathSegments(path string) []string {
	var segments []string
	var current strings.Builder
	depth := 0
	for _, r := range path {
		switch r {
		case '[':
			depth++
			current.WriteRune(r)
		case ']':
			depth--
			current.WriteRune(r)
		case '/':
			if depth == 0 {
				segments = append(segments, current.String())
				current.Reset()
				continue
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}

func parseSegment(segment string) (*gnmipb.PathElem, error) {
	name := segment
	key := ""
	if idx := strings.Index(segment, "["); idx != -1 {
		if !strings.HasSuffix(segment, "]") {
			return nil, fmt.Errorf("unterminated key predicate in %q", segment)
		}
		name = segment[:idx]
		key = segment[idx+1 : len(segment)-1]
	}
	if name == "" {
		return nil, fmt.Errorf("empty path element in %q", segment)
	}

	elem := &gnmipb.PathElem{Name: name}
	if key != "" {
		k, v, ok := strings.Cut(key, "=")
		if !ok {
			return nil, fmt.Errorf("malformed key predicate %q, want key=value", key)
		}
		elem.Key = map[string]string{k: v}
	}
	return elem, nil
}

// PathToString renders a gnmipb.Path back to the "/a/b[k=v]/c" form used
// as the telemetry cache's key, and in log output.
func PathToString(p *gnmipb.Path) string {
	if p == nil || len(p.Elem) == 0 {
		return "/"
	}
	var b strings.Builder
	for _, elem := range p.Elem {
		b.WriteByte('/')
		b.WriteString(elem.Name)
		// Deterministic key order matters for using the string as a map
		// key / log field; sort so the same path always renders the same.
		keys := make([]string, 0, len(elem.Key))
		for k := range elem.Key {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			b.WriteByte('[')
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(elem.Key[k])
			b.WriteByte(']')
		}
	}
	return b.String()
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// ValueToInterface converts a gNMI TypedValue into a plain Go value
// suitable for the telemetry cache and for structured audit logging.
func ValueToInterface(tv *gnmipb.TypedValue) interface{} {
	if tv == nil {
		return nil
	}
	switch v := tv.Value.(type) {
	case *gnmipb.TypedValue_StringVal:
		return v.StringVal
	case *gnmipb.TypedValue_IntVal:
		return v.IntVal
	case *gnmipb.TypedValue_UintVal:
		return v.UintVal
	case *gnmipb.TypedValue_BoolVal:
		return v.BoolVal
	case *gnmipb.TypedValue_FloatVal:
		return v.FloatVal
	case *gnmipb.TypedValue_DoubleVal:
		return v.DoubleVal
	case *gnmipb.TypedValue_DecimalVal:
		return decimalToFloat(v.DecimalVal)
	case *gnmipb.TypedValue_JsonVal:
		return string(v.JsonVal)
	case *gnmipb.TypedValue_JsonIetfVal:
		return string(v.JsonIetfVal)
	case *gnmipb.TypedValue_AsciiVal:
		return v.AsciiVal
	case *gnmipb.TypedValue_LeaflistVal:
		vals := make([]interface{}, 0, len(v.LeaflistVal.GetElement()))
		for _, e := range v.LeaflistVal.GetElement() {
			vals = append(vals, ValueToInterface(e))
		}
		return vals
	default:
		return fmt.Sprintf("%v", tv.Value)
	}
}

func decimalToFloat(d *gnmipb.Decimal64) float64 {
	if d == nil {
		return 0
	}
	f, _ := strconv.ParseFloat(strconv.FormatInt(d.Digits, 10), 64)
	for i := uint32(0); i < d.Precision; i++ {
		f /= 10
	}
	return f
}
