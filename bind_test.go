package zerohttp

import (
	"strings"
	"testing"
)

func TestBinder_JSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid JSON",
			json:      `{"name": "John", "age": 30}`,
			wantError: false,
		},
		{
			name:      "invalid JSON",
			json:      `{"invalid": json}`,
			wantError: true,
		},
		{
			name:      "unknown field",
			json:      `{"name": "John", "unknown": "field"}`,
			wantError: true,
			errorMsg:  "unknown field",
		},
		{
			name:      "empty JSON object",
			json:      `{}`,
			wantError: false,
		},
		{
			name:      "null values",
			json:      `{"name": null, "age": null}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.json)

			var result struct {
				Name *string `json:"name"`
				Age  *int    `json:"age"`
			}

			err := B.JSON(reader, &result)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Verify valid cases
			if tt.name == "valid JSON" {
				if result.Name == nil || *result.Name != "John" {
					t.Errorf("expected name 'John', got %v", result.Name)
				}
				if result.Age == nil || *result.Age != 30 {
					t.Errorf("expected age 30, got %v", result.Age)
				}
			}
		})
	}
}
