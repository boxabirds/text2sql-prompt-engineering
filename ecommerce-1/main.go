package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const DEFAULT_OPENAI_MODEL = "gpt-4-preview"
const DEFAULT_OLLAMA_MODEL = "llama3:instruct"
const OLLAMA_API_KEY = "ollama"
const NO_SEED = -1
const SYSTEM_PROMPT_INSTRUCTION = `
	You are a READ ONLY SQL SELECT Statement Generator API for the schema below ONLY. 
	Generate only queries that access data, not modify it: 
	no UPDATE, INSERT, DELETE or any other statements that attempt to change the data. 
	Respond to questions in a way that can be interpreted programmatically: 
	no extra narrative, punctuation or delimiters.\n`

func createOpenAiClient(baseUrl string, model *string) *openai.Client {

	var apiKey string
	if baseUrl == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("OPENAI_API_KEY not found in environment")
		}
		if *model == "" {
			*model = DEFAULT_OPENAI_MODEL
		}
		fmt.Printf("Using default OpenAI API server\n")
	} else {
		fmt.Printf("=== NOTE AS OF 28 May 2024 Ollama does not appear to use the seed to make output deterministic.===")
		apiKey = OLLAMA_API_KEY
		if *model == "" {
			*model = DEFAULT_OLLAMA_MODEL
		}
		fmt.Printf("Using custom API server at: %s\n", baseUrl)
		fmt.Printf("API Key set to Ollama\n")
	}
	fmt.Printf("Model being used: %s\n", *model)

	config := openai.DefaultConfig(apiKey)

	// have to check twice because the config that's created and depends on it
	// and yet needs to be changed again
	if baseUrl != "" {
		config.BaseURL = baseUrl
	}

	return openai.NewClientWithConfig(config)
}

func rows2String(rows *sql.Rows) (string, error) {
	// Retrieve column names
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	// Prepare slices for scanning
	colVals := make([]interface{}, len(cols))
	scanArgs := make([]interface{}, len(colVals))
	for i := range colVals {
		scanArgs[i] = &colVals[i]
	}

	// Initialize a slice to hold all rows
	allRows := make([]map[string]interface{}, 0)

	// Print the header
	header := make([]string, len(cols))
	copy(header, cols)
	headerLine := strings.Join(header, "\t") + "\n"
	fmt.Print(headerLine)

	// Iterate over rows
	for rows.Next() {
		rowMap := make(map[string]interface{})
		if err := rows.Scan(scanArgs...); err != nil {
			return "", err
		}

		for i, col := range cols {
			rowMap[col] = colVals[i]
		}

		allRows = append(allRows, rowMap)
	}

	// Handle any errors from iterating over rows
	if err := rows.Err(); err != nil {
		return "", err
	}

	// Convert allRows to a string representation
	var sb strings.Builder
	sb.WriteString(fmt.Sprint(allRows))

	return sb.String(), nil
}

func main() {
	db, err := initialiseDb("ecommerce-autogen.db")
	if err != nil {
		fmt.Println(err)
	}
	var seed int

	model := flag.String("model", "", "Model to use for the API")
	baseURL := flag.String("base-url", "", "Base URL for the API server")
	query := flag.String("query", "", "Query to use in against db")
	maxTokens := flag.Int("max-tokens", 200, "Maximum number of tokens in the summary")
	flag.IntVar(&seed, "seed", NO_SEED, "Seed for deterministic results (optional)")

	flag.Parse()

	client := createOpenAiClient(*baseURL, model)

	systemPrompt := SYSTEM_PROMPT_INSTRUCTION + strings.Join(TABLES, "\n")

	req := openai.ChatCompletionRequest{
		Model:       *model,
		MaxTokens:   *maxTokens,
		Temperature: 0.0,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: *query},
		},
	}
	if seed != NO_SEED {
		fmt.Printf("Using fixed seed: %d\n", seed)
		req.Seed = &seed
	}

	start := time.Now()
	ctx := context.Background()
	response, err := client.CreateChatCompletion(ctx, req)
	elapsed := time.Since(start)
	fmt.Printf("Total Execution Time: %s\n", elapsed)
	if err != nil {
		log.Fatalf("ChatCompletion error: %v\n", err)
	}

	sqlQuery := response.Choices[0].Message.Content
	fmt.Printf("SQL Query:\n%s\n", sqlQuery)

	// send query to db
	rows, err := db.Query(sqlQuery)
	if err != nil {
		log.Fatalf("Query error: %v\n", err)
	}
	// print out the rows
	sqlRows, err := rows2String(rows)
	if err != nil {
		log.Fatalf("rows2String error: %v\n", err)
	}
	fmt.Printf("SQL Rows:\n%s\n", sqlRows)
}
