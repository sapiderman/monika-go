// Package assertion parses alert assertion strings into typed, validated predicates.
// A typo like "response.stats != 200" is caught at config-load time, not at runtime.
package assertion

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ProbeResult carries the response fields that assertions can reference.
type ProbeResult struct {
	Err          error             // nil if request succeeded
	Status       int               // HTTP status code
	ResponseTime int64             // round-trip time in milliseconds
	BodySize     int64             // response body size in bytes
	Headers      map[string]string // lowercase keys
	Body         string            // response body text
}

// lhs identifies the left-hand-side field.
type lhs uint8

const (
	lhsStatus lhs = iota
	lhsTime
	lhsSize
	lhsBody
	lhsHeader
)

// op identifies the comparison operator.
type op uint8

const (
	opEq op = iota
	opNe
	opLt
	opGt
	opLte
	opGte
)

func (o op) String() string {
	return [...]string{"==", "!=", "<", ">", "<=", ">="}[o]
}

// rhs tags the right-hand-side literal type.
type rhsKind uint8

const (
	rhsNumber rhsKind = iota
	rhsString
)

// Assertion is a parsed, validated assertion expression.
// Zero value is invalid; use Parse to construct.
type Assertion struct {
	raw   string
	field lhs
	op    op
	kind  rhsKind
	num   int64
	str   string
	hKey  string // non-empty only when field == lhsHeader
}

// Parse compiles an assertion string into a validated Assertion.
// Returns a descriptive error for unknown fields, invalid operators, and type mismatches.
func Parse(expr string) (*Assertion, error) {
	lhsStr, opStr, rhsStr, err := lex(expr)
	if err != nil {
		return nil, err
	}

	field, hKey, err := parseLHS(lhsStr)
	if err != nil {
		return nil, err
	}

	operator, err := parseOp(opStr)
	if err != nil {
		return nil, err
	}

	kind, num, str, err := parseRHS(rhsStr)
	if err != nil {
		return nil, err
	}

	a := &Assertion{raw: expr, field: field, op: operator, kind: kind, num: num, str: str, hKey: hKey}

	err = a.typeCheck()
	if err != nil {
		return nil, err
	}

	return a, nil
}

// MustParse is like Parse but panics on error. Use in tests only.
func MustParse(expr string) *Assertion {
	a, err := Parse(expr)
	if err != nil {
		panic(fmt.Sprintf("assertion: MustParse(%q): %v", expr, err))
	}
	return a
}

// Evaluate returns true when the assertion holds for the given probe result.
func (a *Assertion) Evaluate(r ProbeResult) bool {
	if r.Err != nil {
		return false
	}
	switch a.field {
	case lhsStatus:
		return cmpInt(int64(r.Status), a.op, a.num)
	case lhsTime:
		return cmpInt(r.ResponseTime, a.op, a.num)
	case lhsSize:
		return cmpInt(r.BodySize, a.op, a.num)
	case lhsBody:
		return cmpStr(r.Body, a.op, a.str)
	case lhsHeader:
		return cmpStr(r.Headers[a.hKey], a.op, a.str)
	}
	return false
}

// String returns the original expression for logging and error messages.
func (a *Assertion) String() string { return a.raw }

// UnmarshalYAML implements yaml.Unmarshaler so Alert.Assertion compiles during decode.
func (a *Assertion) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*a = *parsed
	return nil
}

// --- internal helpers ---------------------------------------------------------

func lex(expr string) (string, string, string, error) {
	for _, candidate := range []string{"<=", ">=", "!=", "==", "<", ">"} {
		idx := strings.Index(expr, candidate)
		if idx < 0 {
			continue
		}
		lhsStr := strings.TrimSpace(expr[:idx])
		rhsStr := strings.TrimSpace(expr[idx+len(candidate):])
		if lhsStr == "" || rhsStr == "" {
			return "", "", "", errors.New("incomplete expression")
		}
		return lhsStr, candidate, rhsStr, nil
	}
	return "", "", "", errors.New("no valid operator found (supported: ==, !=, <, >, <=, >=)")
}

func parseLHS(s string) (lhs, string, error) {
	switch s {
	case "response.status":
		return lhsStatus, "", nil
	case "response.time":
		return lhsTime, "", nil
	case "response.size":
		return lhsSize, "", nil
	case "response.body":
		return lhsBody, "", nil
	}
	if strings.HasPrefix(s, `response.headers["`) && strings.HasSuffix(s, `"]`) {
		key := s[len(`response.headers["`) : len(s)-2]
		if key == "" {
			return 0, "", errors.New("header key must not be empty")
		}
		return lhsHeader, strings.ToLower(key), nil
	}
	return 0, "", fmt.Errorf("unknown field %q (valid: "+
		"response.status, response.time, response.size, response.body, "+
		"response.headers[\"key\"])", s)
}

func parseOp(s string) (op, error) {
	switch s {
	case "==":
		return opEq, nil
	case "!=":
		return opNe, nil
	case "<":
		return opLt, nil
	case ">":
		return opGt, nil
	case "<=":
		return opLte, nil
	case ">=":
		return opGte, nil
	}
	return 0, fmt.Errorf("unknown operator %q", s)
}

func parseRHS(s string) (rhsKind, int64, string, error) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return rhsNumber, n, "", nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return rhsNumber, int64(f), "", nil
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		return rhsString, 0, s[1 : len(s)-1], nil
	}
	return 0, 0, "", fmt.Errorf("unrecognised literal %q; expected number or quoted string", s)
}

func (a *Assertion) typeCheck() error {
	// Ordered operators require numeric LHS and numeric RHS.
	isOrdered := a.op == opLt || a.op == opGt || a.op == opLte || a.op == opGte

	if a.kind == rhsString && a.field != lhsBody && a.field != lhsHeader {
		return fmt.Errorf("string literal not valid for %s field", fieldLabel(a.field))
	}
	if isOrdered && (a.field == lhsBody || a.field == lhsHeader) {
		return fmt.Errorf("operator %s not valid for string field", a.op)
	}
	return nil
}

func fieldLabel(f lhs) string {
	switch f {
	case lhsStatus:
		return "response.status"
	case lhsTime:
		return "response.time"
	case lhsSize:
		return "response.size"
	case lhsBody:
		return "response.body"
	case lhsHeader:
		return "response.headers"
	}
	return "unknown"
}

func cmpInt(got int64, operator op, rhs int64) bool {
	switch operator {
	case opEq:
		return got == rhs
	case opNe:
		return got != rhs
	case opLt:
		return got < rhs
	case opGt:
		return got > rhs
	case opLte:
		return got <= rhs
	case opGte:
		return got >= rhs
	}
	return false
}

func cmpStr(got string, operator op, rhs string) bool {
	switch operator {
	case opEq:
		return got == rhs
	case opNe:
		return got != rhs
	case opLt, opGt, opLte, opGte:
		return false
	default:
		return false
	}
}
