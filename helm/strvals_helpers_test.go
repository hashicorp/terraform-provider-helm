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

func TestCloakSetValue(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]interface{}
		key    string
		get    func(map[string]interface{}) interface{}
	}{
		{
			name:   "plain key",
			values: map[string]interface{}{"password": "secret"},
			key:    "password",
			get:    func(m map[string]interface{}) interface{} { return m["password"] },
		},
		{
			name: "dotted key",
			values: map[string]interface{}{
				"auth": map[string]interface{}{"password": "secret"},
			},
			key: "auth.password",
			get: func(m map[string]interface{}) interface{} {
				return m["auth"].(map[string]interface{})["password"]
			},
		},
		{
			name:   "escaped-dotted key",
			values: map[string]interface{}{"foo.bar": "secret"},
			key:    `foo\.bar`,
			get:    func(m map[string]interface{}) interface{} { return m["foo.bar"] },
		},
		{
			name: "nested key",
			values: map[string]interface{}{
				"server": map[string]interface{}{
					"config": map[string]interface{}{
						"oidc.config": "my-secret",
					},
				},
			},
			key: `server.config.oidc\.config`,
			get: func(m map[string]interface{}) interface{} {
				return m["server"].(map[string]interface{})["config"].(map[string]interface{})["oidc.config"]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloakSetValue(tt.values, tt.key)
			if got := tt.get(tt.values); got != sensitiveContentValue {
				t.Errorf("key %q: expected %q, got %q", tt.key, sensitiveContentValue, got)
			}
		})
	}
}
