package remilia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func TestGetOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		def      string
		expected string
	}{
		{"Empty string with default", new(string), "default", "default"},
		{"Non-empty string", strPtr("non-empty"), "default", "non-empty"},
		{"Nil pointer with default", nil, "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetOrDefault(tt.input, tt.def)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestURLMather(t *testing.T) {
	matcher := URLMatcher()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid HTTP URL", "http://example.com", true},
		{"Valid HTTPS URL", "https://example.com", true},
		{"Valid FTP URL", "ftp://example.com", true},
		{"Invalid URL", "not_a_url", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher(tt.input)
			if result != tt.expected {
				t.Errorf("URLMatcher()(%q) = %t, want %t", tt.input, result, tt.expected)
			}
		})
	}
}
