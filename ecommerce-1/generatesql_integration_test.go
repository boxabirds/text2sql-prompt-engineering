package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

const LocalOllamaBaseUrl = "http://localhost:11434/v1"

func TestCompareSqlQueriesNonExactMatch(t *testing.T) {
	fmt.Printf("\n\n### WARNING this integration test relies on llama3 being present on a local ollama instance\n\n")
	clients := initialiseLLMClients(LocalOllamaBaseUrl)
	evaluationClient := getLLMClient("Ollama/OpenAI", "llama3", clients)
	// if evaluationClient is nil we need to fail the test
	if evaluationClient == nil {
		log.Fatal("Failed to initialise evaluation client")
	}
	assert.NotEqual(t, nil, evaluationClient)
	maxTokens := 100
	seed := 42

	// Scenario 2: Functional match due to alias difference
	result1, err1 := compareSqlQueries("SELECT p.name FROM products p", "SELECT prod.name FROM products prod", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err1)
	assert.Equal(t, FunctionalMatch, result1)

	// Scenario 3: Functional superset match
	result2, err2 := compareSqlQueries("SELECT product_name FROM products", "SELECT product_name, product_price FROM products", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err2)
	assert.Equal(t, FunctionalSupersetMatch, result2)

	// Scenario 4: None match
	result3, err3 := compareSqlQueries("SELECT name FROM users", "SELECT age FROM users", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err3)
	assert.Equal(t, NoMatch, result3)
}
