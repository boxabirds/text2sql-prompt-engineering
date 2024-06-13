package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/tmc/langchaingo/llms"
)

const SqlGeneratorApiSystemPrompt = `
	You are a READ ONLY SQL SELECT Statement Generator API for the schema below ONLY. 
	Generate only queries that access data, not modify it: 
	no UPDATE, INSERT, DELETE or any other statements that attempt to change the data. 
	Respond to questions in a way that can be interpreted programmatically: 
	no extra narrative, punctuation, delimiters or escape sequences like backticks.\n`

const SqlComparisonApiSystemPrompt = `
	You are a SQL Statement comparator API: Take two SQL queries, a ground truth and a comparision, and compare them to determine
	how similar they are, returning only a single word from this list: "None", "FunctionalSuperset", or "Functional"
	
	Rules for returning the value "Functional": BOTH rules 1 and 2 below:

	1. Any difference in interim join aliases can be ignored as they do not affect output.
	Example 1: "op" can be any text in this query and it'll be FunctionalMatch: SELECT p."name", SUM(op."quantity" * p."price") AS "profit" FROM "Order_Products" op JOIN "Products" p ON op."product_id" = p."id" GROUP BY p."name" ORDER BY "profit" DESC LIMIT 1;
	
	2. The output column names can vary from ground truth query and comparison query if they're semantically equivalent. 
	E.g. for an order query, Query 1: SELECT "order_value" and Query 2: SELECT "total_order_value" are semantically equivalent because total_value and total_order_value in the context of an order query are equivalent.
	
	Rules for "FunctionalSuperset": rules 1 and 2 AND 3 below:
	3. If the output columns of the comparison query contain all the columns of the ground truth but some additional columns,
	then the comparison query is a functional superset of the ground truth query and the value "FunctionalSuperset" should be returned.
	E.g. Query 1: SELECT "product_name", Query 2: SELECT "product_name", "product_price"

	Rules for "None": output columns of comparison query is missing one or more columns included in the ground truth query

	Respond to questions in a way that can be interpreted programmatically: 
	NO extra narrative, punctuation, delimiters or escape sequences like backticks.\n\n
`
const QueryPromptTemplate = `
	Ground truth sql statement: {{.GroundTruthQuery}}\n
	Comparison sql query: {{.ComparisonQuery}}`

const MaxSqlGenerationFaultRetries = 2

type SqlQueryEvaluationType string

const (
	NoMatch                 SqlQueryEvaluationType = "None"
	FunctionalSupersetMatch SqlQueryEvaluationType = "FunctionalSuperset" // second query has everything the first query has with extra fields that can be ignored
	FunctionalMatch         SqlQueryEvaluationType = "Functional"         // sql queries might not have the same output columns but the the columns have the same meaning
	ExactMatch              SqlQueryEvaluationType = "Exact"              // sql queries are character by character identical
)

func substituteTemplate(promptTemplate string, params map[string]string) string {
	// Parse the template string
	//promptTemplate = "Hello, {{.Name}}!"
	tmpl, err := template.New("queryPrompt").Parse(promptTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}
	tmpl.Option("missingkey=zero")
	// Execute the template with parameters
	var substituted bytes.Buffer
	if err := tmpl.Execute(&substituted, params); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	return substituted.String()
}

// Takes a ground truth sql query and a comparison sql query and uses the evaluator
// to appropriate match.
func compareSqlQueries(groundTruthSqlQuery string, comparisonQuery string, evaluatorLLM *LLMClient, maxTokens *int, seed int) (SqlQueryEvaluationType, error) {

	if evaluatorLLM == nil {
		log.Fatal("evaluatorLLM cannot be nil")
	}
	// query2 is exactly the same as query1 which makes life easy
	if groundTruthSqlQuery == comparisonQuery {
		return ExactMatch, nil
	}
	ctx := context.Background()
	options := []llms.CallOption{
		llms.WithMaxTokens(*maxTokens),
		llms.WithTemperature(0.0),
	}
	if seed != NoSeed {
		options = append(options, llms.WithSeed(seed))
	}

	// Substitute and print the result
	comparisonPrompt := substituteTemplate(QueryPromptTemplate, map[string]string{
		"GroundTruthQuery": groundTruthSqlQuery,
		"ComparisonQuery":  comparisonQuery,
	})

	start := time.Now()
	response, err := llms.GenerateFromSinglePrompt(ctx, evaluatorLLM.Instance, SqlComparisonApiSystemPrompt+comparisonPrompt, options...)
	elapsed := time.Since(start)
	fmt.Printf("- compareSqlQueries generation execution time: %s\n", elapsed)

	if err != nil {
		return "", err
	}
	fmt.Printf("Response: '%s'\n", response)
	return SqlQueryEvaluationType(response), nil

}

func predictSqlQueryFromNaturalLanguageQuery(llm llms.Model, maxTokens *int, systemPrompt string, query *string, seed int, failedAttempts []FailedSqlQueryAttempt) (string, error) {
	// Modify the system prompt to include the history of failed attempts
	//fmt.Printf("- Query: '%s'\n", *query)
	if len(failedAttempts) > 0 {
		systemPrompt += "\nTake into account the following past failed attempts at generating a new SQL query that avoids the same errors:\n"
		for _, attempt := range failedAttempts {
			systemPrompt += fmt.Sprintf("Generated failed sql query: '%s';\nError message explaining why it failed:\n'%s'\n", strings.ReplaceAll(attempt.SqlQuery, "\n", " "), strings.ReplaceAll(attempt.ErrorMessage, "\n", " "))
		}
	}

	// print out system prompt
	//fmt.Printf("- System Prompt:\n--------\n%s\n--------\n", strings.ReplaceAll(systemPrompt, "\n", " "))

	ctx := context.Background()
	options := []llms.CallOption{
		llms.WithMaxTokens(*maxTokens),
		llms.WithTemperature(0.0),
	}
	if seed != NoSeed {
		options = append(options, llms.WithSeed(seed))
	}

	start := time.Now()
	response, err := llms.GenerateFromSinglePrompt(ctx, llm, systemPrompt+"\n"+*query, options...)
	elapsed := time.Since(start)
	fmt.Printf("- Query generation execution time: %s\n", elapsed)

	if err != nil {
		return "", err
	}

	return response, nil
}
