package template

import "testing"

func TestRenderBody(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		payload  map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "simple substitution",
			tmpl:     `{"user": "{{.name}}", "event": "{{.event}}"}`,
			payload:  map[string]interface{}{"name": "alice", "event": "signup"},
			expected: `{"user": "alice", "event": "signup"}`,
		},
		{
			name:     "empty template",
			tmpl:     "",
			payload:  map[string]interface{}{"name": "alice"},
			expected: "",
		},
		{
			name:     "no placeholders",
			tmpl:     `{"static": "value"}`,
			payload:  map[string]interface{}{},
			expected: `{"static": "value"}`,
		},
		{
			name:    "invalid template syntax",
			tmpl:    `{{.unclosed`,
			payload: map[string]interface{}{},
			wantErr: true,
		},
		{
			name:     "numeric value",
			tmpl:     `{"amount": {{.amount}}}`,
			payload:  map[string]interface{}{"amount": 99.5},
			expected: `{"amount": 99.5}`,
		},
		{
			name:     "missing key renders zero value",
			tmpl:     `{"val": "{{.missing}}"}`,
			payload:  map[string]interface{}{},
			expected: `{"val": "<no value>"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderBody(tt.tmpl, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
