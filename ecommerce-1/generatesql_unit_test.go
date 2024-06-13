package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareSqlQueriesExactMatch(t *testing.T) {
	client := &LLMClient{}
	maxTokens := 100
	seed := 42

	// Scenario 1: Exact match
	result, err := compareSqlQueries("SELECT name FROM users", "SELECT name FROM users", client, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, ExactMatch, result)
}

func TestSubstituteTemplate(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		params   map[string]string
		expected string
	}{
		{
			name:     "Simple substitution",
			template: "Hello, {{.Name}}!",
			params:   map[string]string{"Name": "World"},
			expected: "Hello, World!",
		},
		{
			name:     "Multiple substitutions",
			template: "User: {{.Username}}, Email: {{.Email}}",
			params:   map[string]string{"Username": "jdoe", "Email": "jdoe@example.com"},
			expected: "User: jdoe, Email: jdoe@example.com",
		},
		{
			name:     "Missing parameter",
			template: "Hello, {{.Name}}!",
			params:   map[string]string{},
			expected: "Hello, !",
		},
		{
			name:     "Extra whitespace around values",
			template: "Hello, {{.Name}}!",
			params:   map[string]string{"Name": "   World   "},
			expected: "Hello,    World   !",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := substituteTemplate(tc.template, tc.params)
			assert.Equal(t, tc.expected, actual, "They should be equal")
		})
	}
}
