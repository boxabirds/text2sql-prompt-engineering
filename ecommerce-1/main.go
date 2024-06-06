package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const GROUND_TRUTH_MD_FILE = "ground-truth.md"

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

type GroundTruthItem struct {
	Query  string
	SQL    string
	Result string
}

func loadGroundTruthCsv(filename string) ([]GroundTruthItem, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var groundTruth []GroundTruthItem

	// Skip the header row if your CSV has headers
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 3 {
			continue // skip rows that do not have enough columns
		}

		item := GroundTruthItem{
			Query:  record[0],
			SQL:    record[1],
			Result: record[2],
		}
		groundTruth = append(groundTruth, item)
	}

	return groundTruth, nil
}
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
	// deal with command line flags first
	model := flag.String("model", "", "Model to use for the API")
	baseURL := flag.String("base-url", "", "Base URL for the API server")
	//query := flag.String("query", "", "Query to use in against db")
	maxTokens := flag.Int("max-tokens", 200, "Maximum number of tokens in the summary")
	var seed int
	flag.IntVar(&seed, "seed", NO_SEED, "Seed for deterministic results (optional)")
	flag.Parse()

	// ensure our db exists and has the content we want to test against
	db, err := initialiseDb("ecommerce-autogen.db")
	if err != nil {
		fmt.Println(err)
	}

	// ensure we have our ground truth MD file in a CSV file for easy processing
	groundTruthCsvFile, err := convertMdWithSingleTableToCsv(GROUND_TRUTH_MD_FILE)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Ensured we have a ground truth csv in '%s'\n", groundTruthCsvFile)

	// load the ground truth
	groundTruth, err := loadGroundTruthCsv(groundTruthCsvFile)
	if err != nil {
		log.Fatalf("Failed to load CSV: %v", err)
	}

	fmt.Printf("Loaded %d ground truth items\n", len(groundTruth))

	// do the AI stuff to predict the SQL query from natural language
	client := createOpenAiClient(*baseURL, model)
	systemPrompt := SYSTEM_PROMPT_INSTRUCTION + strings.Join(TABLES, "\n")

	for _, item := range groundTruth {
		// Generate predicted SQL query
		predictedQuery, err := predictSqlQueryFromNaturalLanguageQuery(model, maxTokens, systemPrompt, &item.Query, seed, client)
		if err != nil {
			log.Printf("Error predicting SQL for query '%s': %v\n", item.Query, err)
			continue
		}

		// Compare ground truth with predicted query
		match := "different"
		if item.SQL == predictedQuery {
			match = "same"
		}

		fmt.Printf("====\n")
		fmt.Printf("Ground Truth Query: '%s'\nPredicted Query: '%s'\nResult: %s\n\n", item.SQL, predictedQuery, match)
		sqlRows, err := runQuery(db, item.SQL)
		if err != nil {
			log.Printf("Error executing query '%s': %v", item.SQL, err)
			continue
		}

		fmt.Printf("SQL Rows:\n%s\n", sqlRows)
	}
}

func runQuery(db *sql.DB, sqlQuery string) (string, error) {
	rows, err := db.Query(sqlQuery)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	sqlRows, err := rows2String(rows)
	if err != nil {
		return "", err
	}
	return sqlRows, nil
}

func predictSqlQueryFromNaturalLanguageQuery(model *string, maxTokens *int, systemPrompt string, query *string, seed int, client *openai.Client) (string, error) {
	fmt.Printf("Query: %s\n", *query)
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
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	sqlQuery := response.Choices[0].Message.Content
	fmt.Printf("SQL Query:\n%s\n", sqlQuery)
	return sqlQuery, nil
}
