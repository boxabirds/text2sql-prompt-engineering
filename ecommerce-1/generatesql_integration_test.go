package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

// probably need to change this to localhost for most people
const LocalOllamaBaseUrl = "http://gruntus:11434/v1"

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

	var result SqlQueryEvaluationType
	var err error
	var groundTruthSqlQuery string
	var comparisonSqlQuery string

	// Scenario 2: Functional match due to alias difference
	result, err = compareSqlQueries("SELECT p.name FROM products p", "SELECT prod.name FROM products prod", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, FunctionalMatch, result)

	// Missing output column
	groundTruthSqlQuery = `SELECT c."name", SUM(op."quantity" * p."price") AS "profit" FROM "Order_Products" op JOIN "Orders" o ON op."order_id" = o."id" JOIN "Customers" c ON o."customer_id" = c."id" JOIN "Products" p ON op."product_id" = p."id" GROUP BY c."name" ORDER BY "profit" DESC LIMIT 1;`
	comparisonSqlQuery = `SELECT SUM(op."quantity" * p."price") AS "profit" FROM "Order_Products" op JOIN "Orders" o ON op."order_id" = o."id" JOIN "Customers" c ON o."customer_id" = c."id" JOIN "Products" p ON op."product_id" = p."id" GROUP BY c."name" ORDER BY "profit" DESC LIMIT 1;`
	assert.NoError(t, err)
	assert.Equal(t, NoMatch, result)

	// Scenario 3: Functional superset match
	result, err = compareSqlQueries("SELECT product_name FROM products", "SELECT product_name, product_price FROM products", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, FunctionalMatch, result)

	result, err = compareSqlQueries("SELECT COUNT(*) FROM \"Customers\";", "SELECT COUNT(*) FROM Customers;", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, FunctionalMatch, result)

	// 'SELECT SUM("quantity") AS "total_sold" FROM "Order_Products" WHERE "product_id" = (SELECT "id" FROM "Products" WHERE "name" = 'Product 7');'
	// ' SELECT SUM(quantity) FROM Order_Products JOIN Products ON Order_Products.product_id = Products.id WHERE name = 'Product 7';'
	groundTruthSqlQuery = `SELECT SUM("quantity") AS "total_sold" FROM "Order_Products" WHERE "product_id" = (SELECT "id" FROM "Products" WHERE "name" = 'Product 7');`
	comparisonSqlQuery = `SELECT SUM(quantity) FROM Order_Products JOIN Products ON Order_Products.product_id = Products.id WHERE name = 'Product 7';`
	result, err = compareSqlQueries(groundTruthSqlQuery, comparisonSqlQuery, evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, FunctionalMatch, result)

	groundTruthSqlQuery = `SELECT SUM(op."quantity" * p."price") AS "total_value" FROM "Order_Products" op JOIN "Orders" o ON op."order_id" = o."id" JOIN "Products" p ON op "product_id" = p."id";`
	comparisonSqlQuery = ` SELECT SUM(Products.price * Order_Products.quantity) AS TotalValueOfOrders FROM Orders JOIN Order_Products ON Orders.id = Order_Products.order_id JOIN Products ON Order_Products.product_id = Products.id;`
	result, err = compareSqlQueries(groundTruthSqlQuery, comparisonSqlQuery, evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, FunctionalMatch, result)

	// Scenario 4: None match
	result, err = compareSqlQueries("SELECT name FROM users", "SELECT age FROM users", evaluationClient, &maxTokens, seed)
	assert.NoError(t, err)
	assert.Equal(t, NoMatch, result)
}
