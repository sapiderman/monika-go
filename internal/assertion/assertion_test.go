//nolint:testpackage // needs to test internal/unexported components
package assertion

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParse(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"response.status == 200", false},
		{"response.status != 200", false},
		{"response.time > 1000", false},
		{"response.time >= 1000", false},
		{"response.time < 1000", false},
		{"response.time <= 1000", false},
		{`response.body == "ok"`, false},
		{`response.body != "error"`, false},
		{`response.headers["content-type"] == "application/json"`, false},
		{"response.size == 1024", false},
		// errors
		{"", true},                       // empty
		{"response.stats != 200", true},  // typo
		{"response.status ~= 200", true}, // bad operator
		{`response.body > "ok"`, true},   // > on string
		{"response.status == ok", true},  // non-numeric rhs for int field
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			a, err := Parse(tt.expr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.expr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.expr, err)
				return
			}
			if a.String() != tt.expr {
				t.Errorf("String() = %q, want %q", a.String(), tt.expr)
			}
		})
	}
}

func TestEvaluate_Numeric(t *testing.T) {
	tests := []struct {
		expr   string
		result ProbeResult
		want   bool
	}{
		{"response.status == 200", ProbeResult{Status: 200}, true},
		{"response.status == 200", ProbeResult{Status: 500}, false},
		{"response.status != 200", ProbeResult{Status: 500}, true},
		{"response.status != 200", ProbeResult{Status: 200}, false},
		{"response.time > 100", ProbeResult{ResponseTime: 200}, true},
		{"response.time > 100", ProbeResult{ResponseTime: 50}, false},
		{"response.time >= 100", ProbeResult{ResponseTime: 100}, true},
		{"response.time < 100", ProbeResult{ResponseTime: 50}, true},
		{"response.time <= 100", ProbeResult{ResponseTime: 100}, true},
		{"response.size == 1024", ProbeResult{BodySize: 1024}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			a := MustParse(tt.expr)
			got := a.Evaluate(tt.result)
			if got != tt.want {
				t.Errorf("Evaluate(%+v) = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

func TestEvaluate_String(t *testing.T) {
	tests := []struct {
		expr   string
		result ProbeResult
		want   bool
	}{
		{`response.body == "ok"`, ProbeResult{Body: "ok"}, true},
		{`response.body == "ok"`, ProbeResult{Body: "error"}, false},
		{`response.body != "error"`, ProbeResult{Body: "ok"}, true},
		{
			`response.headers["content-type"] == "application/json"`,
			ProbeResult{Headers: map[string]string{"content-type": "application/json"}},
			true,
		},
		{
			`response.headers["content-type"] != "text/html"`,
			ProbeResult{Headers: map[string]string{"content-type": "application/json"}},
			true,
		},
		{
			`response.headers["x-missing"] == "nothing"`,
			ProbeResult{Headers: map[string]string{}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			a := MustParse(tt.expr)
			got := a.Evaluate(tt.result)
			if got != tt.want {
				t.Errorf("Evaluate(%+v) = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid expression")
		}
	}()
	MustParse("response.stats != 200")
}

func TestEvaluate_WithError(t *testing.T) {
	tests := []struct {
		expr   string
		result ProbeResult
	}{
		{"response.status == 200", ProbeResult{Err: someError{}, Status: 200}},
		{"response.status != 200", ProbeResult{Err: someError{}, Status: 500}},
		{"response.time < 100", ProbeResult{Err: someError{}, ResponseTime: 50}},
		{`response.body == "ok"`, ProbeResult{Err: someError{}, Body: "ok"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			a := MustParse(tt.expr)
			if got := a.Evaluate(tt.result); got {
				t.Errorf("Evaluate(%+v) = %v, want false because of non-nil Err", tt.result, got)
			}
		})
	}
}

type someError struct{}

func (someError) Error() string { return "some error" }

// --- Additional Parse error paths -------------------------------------------

func TestParse_IncompleteExpression(t *testing.T) {
	tests := []string{
		"response.status == ",
		" == 200",
		"",
		"response.status",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := Parse(expr)
			if err == nil {
				t.Errorf("expected error for %q, got nil", expr)
			}
		})
	}
}

func TestParse_EmptyHeaderKey(t *testing.T) {
	_, err := Parse(`response.headers[""] == "value"`)
	if err == nil {
		t.Fatal("expected error for empty header key, got nil")
	}
	if err.Error() != "header key must not be empty" {
		t.Errorf("error = %v, want 'header key must not be empty'", err)
	}
}

func TestParse_OrderedOpOnString(t *testing.T) {
	tests := []string{
		`response.body > "ok"`,
		`response.headers["x"] < "val"`,
		`response.body >= "ok"`,
		`response.headers["x"] <= "val"`,
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := Parse(expr)
			if err == nil {
				t.Errorf("expected error for ordered op on string, got nil")
			}
		})
	}
}

func TestParse_QuotedNumeric(t *testing.T) {
	// "200" is a string literal, not numeric.
	a, err := Parse(`response.body == "200"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.kind != rhsString {
		t.Errorf("expected rhsString kind, got %d", a.kind)
	}
	if a.str != "200" {
		t.Errorf("str = %q, want 200", a.str)
	}
}

func TestParse_FloatLiteral(t *testing.T) {
	a, err := Parse("response.time == 12.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.num != 12 {
		t.Errorf("num = %d, want 12 (truncated from 12.5)", a.num)
	}
}

func TestParse_SingleQuoteString(t *testing.T) {
	a, err := Parse(`response.body == 'hello world'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.str != "hello world" {
		t.Errorf("str = %q, want hello world", a.str)
	}
}

func TestParse_UnclosedQuote(t *testing.T) {
	_, err := Parse(`response.body == "unclosed`)
	if err == nil {
		t.Fatal("expected error for unclosed quote, got nil")
	}
}

// --- Evaluate edge cases -----------------------------------------------------

func TestEvaluate_UnknownField(t *testing.T) {
	// Manually construct an assertion with an invalid field to test default branch.
	a := &Assertion{raw: "x", field: 99, op: opEq, kind: rhsNumber, num: 1}
	result := ProbeResult{Status: 200}
	if a.Evaluate(result) {
		t.Error("expected false for unknown field")
	}
}

func TestEvaluate_StatusEq(t *testing.T) {
	a := MustParse("response.status == 200")
	if !a.Evaluate(ProbeResult{Status: 200}) {
		t.Error("status 200 should match == 200")
	}
	if a.Evaluate(ProbeResult{Status: 500}) {
		t.Error("status 500 should not match == 200")
	}
}

func TestEvaluate_StatusGte(t *testing.T) {
	a := MustParse("response.status >= 200")
	if !a.Evaluate(ProbeResult{Status: 200}) {
		t.Error("200 >= 200 should be true")
	}
	if !a.Evaluate(ProbeResult{Status: 201}) {
		t.Error("201 >= 200 should be true")
	}
	if a.Evaluate(ProbeResult{Status: 199}) {
		t.Error("199 >= 200 should be false")
	}
}

func TestEvaluate_HeaderCaseInsensitive(t *testing.T) {
	// Headers are stored with lowercase keys, but expression key is also lowercased.
	a := MustParse(`response.headers["Content-Type"] == "text/html"`)
	result := ProbeResult{
		Headers: map[string]string{"content-type": "text/html"},
	}
	if !a.Evaluate(result) {
		t.Error("header match should be case-insensitive")
	}
}

func TestEvaluate_HeaderNotFound(t *testing.T) {
	a := MustParse(`response.headers["x-custom"] == "value"`)
	result := ProbeResult{
		Headers: map[string]string{"content-type": "text/html"},
	}
	if a.Evaluate(result) {
		t.Error("missing header should return false")
	}
}

func TestEvaluate_BodyNe(t *testing.T) {
	a := MustParse(`response.body != "error"`)
	if !a.Evaluate(ProbeResult{Body: "ok"}) {
		t.Error("body 'ok' should not equal 'error'")
	}
	if a.Evaluate(ProbeResult{Body: "error"}) {
		t.Error("body 'error' should equal 'error' (!= should fail)")
	}
}

func TestAssertion_String(t *testing.T) {
	expr := "response.status == 200"
	a := MustParse(expr)
	if a.String() != expr {
		t.Errorf("String() = %q, want %q", a.String(), expr)
	}
}

// --- op String() -------------------------------------------------------------

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		want    string
	}{
		{"valid expression", "assertion: response.status == 200", false, "response.status == 200"},
		{"invalid expression", "assertion: bogus", true, ""},
		{"non-string node", "assertion:\n  a: b", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a struct {
				Assertion *Assertion `yaml:"assertion"`
			}
			err := yaml.Unmarshal([]byte(tt.yaml), &a)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if a.Assertion.String() != tt.want {
				t.Errorf("Assertion.String() = %q, want %q", a.Assertion.String(), tt.want)
			}
		})
	}
}

func TestParse_StringLiteralOnIntField(t *testing.T) {
	_, err := Parse(`response.status == "ok"`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not valid for response.status") {
		t.Errorf("expected field label in error, got: %v", err)
	}
}

func TestCmpStr(t *testing.T) {
	tests := []struct {
		name string
		got  string
		op   op
		rhs  string
		want bool
	}{
		{"eq true", "a", opEq, "a", true},
		{"eq false", "a", opEq, "b", false},
		{"ne true", "a", opNe, "b", true},
		{"ne false", "a", opNe, "a", false},
		{"lt unsupported", "a", opLt, "b", false},
		{"gt unsupported", "a", opGt, "b", false},
		{"lte unsupported", "a", opLte, "b", false},
		{"gte unsupported", "a", opGte, "b", false},
		{"unknown op", "a", op(200), "a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmpStr(tt.got, tt.op, tt.rhs); got != tt.want {
				t.Errorf("cmpStr(%q, %v, %q) = %v, want %v", tt.got, tt.op, tt.rhs, got, tt.want)
			}
		})
	}
}

func TestOpString(t *testing.T) {
	tests := []struct {
		opVal op
		want  string
	}{
		{opEq, "=="},
		{opNe, "!="},
		{opLt, "<"},
		{opGt, ">"},
		{opLte, "<="},
		{opGte, ">="},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.opVal.String(); got != tt.want {
				t.Errorf("op.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
