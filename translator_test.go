package main

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a string", "a_string.txt"},
		{"a string with spaces", "a_string_with_s.txt"},
		{"!@#a string_with_special_chars!@#", "___a_string_wit.txt"},
	}

	for _, test := range tests {
		result := sanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFilename(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
