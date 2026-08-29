// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"reflect"
	"testing"
)

func TestSplitKeyPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple single key",
			input:    "simple",
			expected: []string{"simple"},
		},
		{
			name:     "dotted path",
			input:    "a.b.c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "single escaped dot",
			input:    `a\.b`,
			expected: []string{"a.b"},
		},
		{
			name:     "mixed escaped and unescaped dots",
			input:    `server.config.oidc\.config`,
			expected: []string{"server", "config", "oidc.config"},
		},
		{
			name:     "all dots escaped",
			input:    `a\.b\.c`,
			expected: []string{"a.b.c"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "trailing dot",
			input:    "a.",
			expected: []string{"a", ""},
		},
		{
			name:     "leading dot",
			input:    ".a",
			expected: []string{"", "a"},
		},
		{
			name:     "multiple escaped dots no unescaped",
			input:    `no\.dots\.here`,
			expected: []string{"no.dots.here"},
		},
		{
			name:     "backslash not before dot",
			input:    `a\b.c`,
			expected: []string{`a\b`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitKeyPath(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("splitKeyPath(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCloakSetValue_EscapedDot(t *testing.T) {
	values := map[string]interface{}{
		"server": map[string]interface{}{
			"config": map[string]interface{}{
				"oidc.config": "my-secret",
			},
		},
	}

	cloakSetValue(values, `server.config.oidc\.config`)

	got := values["server"].(map[string]interface{})["config"].(map[string]interface{})["oidc.config"]
	if got != sensitiveContentValue {
		t.Errorf("expected %q, got %q", sensitiveContentValue, got)
	}
}

func TestCloakSetValue_NormalDottedPath(t *testing.T) {
	values := map[string]interface{}{
		"auth": map[string]interface{}{
			"password": "secret123",
		},
	}

	cloakSetValue(values, "auth.password")

	got := values["auth"].(map[string]interface{})["password"]
	if got != sensitiveContentValue {
		t.Errorf("expected %q, got %q", sensitiveContentValue, got)
	}
}
