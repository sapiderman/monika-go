package assertion

import "testing"

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
		{"", true},                              // empty
		{"response.stats != 200", true},         // typo
		{"response.status ~= 200", true},         // bad operator
		{`response.body > "ok"`, true},           // > on string
		{"response.status == ok", true},          // non-numeric rhs for int field
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
		{`response.headers["content-type"] == "application/json"`, ProbeResult{Headers: map[string]string{"content-type": "application/json"}}, true},
		{`response.headers["content-type"] != "text/html"`, ProbeResult{Headers: map[string]string{"content-type": "application/json"}}, true},
		{`response.headers["x-missing"] == "nothing"`, ProbeResult{Headers: map[string]string{}}, false},
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