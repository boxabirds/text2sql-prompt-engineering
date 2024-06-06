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
const MAX_RETRIES = 2

type GroundTruthItem struct {
	Query  string
	SQL    string
	Result string
}

type FailedSqlQueryAttempt struct {
	SqlQuery     string
	ErrorMessage string
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
		//fmt.Printf("=== NOTE AS OF 28 May 2024 Ollama does not appear to use the seed to make output deterministic.===")
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
	//headerLine := strings.Join(header, "\t") + "\n"
	//fmt.Print(headerLine)

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

func stripNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
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
		var failedAttempts []FailedSqlQueryAttempt
		var predictedQuery string
		var err error
		fmt.Printf("\n====\n")

		successfulSqlQuery := false

		for len(failedAttempts) <= MAX_RETRIES && !successfulSqlQuery {
			// predict the SQL query from the natural language query
			predictedQuery, err = predictSqlQueryFromNaturalLanguageQuery(client, model, maxTokens, systemPrompt, &item.Query, seed, failedAttempts)
			if err != nil {
				//log.Printf("Error predicting SQL for query '%s': %v\n", item.Query, err)
				continue
			}
			// SQL statements are often multi-line but work on a single line so for readability we compress it to a single line
			predictedQuery = stripNewlines(predictedQuery)

			// Execute the SQL query
			sqlRows, err := runQuery(db, predictedQuery)

			// SQL query failed so let's regenerate the query
			// taking into account this and previous errors
			// by including them in the message sent to the LLM, and try again.
			if err != nil {
				log.Printf("Error executing query so retrying '%s': %v", predictedQuery, err)
				failedAttempts = append(failedAttempts, FailedSqlQueryAttempt{
					// Compress the sql query to a single line
					SqlQuery:     predictedQuery,
					ErrorMessage: stripNewlines(err.Error()),
				})

				// generating the query was successful, so let's compare against ground truth
			} else {
				successfulSqlQuery = true
				match := "different"
				if item.SQL == predictedQuery {
					match = "same"
				}

				fmt.Printf("Ground Truth Query: '%s'\nSuccessfully Executed Predicted Query: '%s'\nResult: %s\n\n", item.SQL, predictedQuery, match)
				fmt.Printf("SQL Rows:\n%s\n", sqlRows)
			}
		}

		if err != nil {
			log.Printf("Failed to execute a valid query after %d attempts for query '%s'.", MAX_RETRIES+1, item.Query)
		}
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

func predictSqlQueryFromNaturalLanguageQuery(client *openai.Client, model *string, maxTokens *int, systemPrompt string, query *string, seed int, failedAttempts []FailedSqlQueryAttempt) (string, error) {
	// Modify the system prompt to include the history of failed attempts
	//fmt.Printf("- Query: '%s'\n", *query)
	if len(failedAttempts) > 0 {
		systemPrompt += "\nTake into account the following past failed attempts at generating a SQL query when creating the query to avoid the same mistakes:\n"
		for _, attempt := range failedAttempts {
			systemPrompt += fmt.Sprintf("Generated failed sql query: '%s'; Error message explaining why it failed: '%s'\n", strings.ReplaceAll(attempt.SqlQuery, "\n", " "), strings.ReplaceAll(attempt.ErrorMessage, "\n", " "))
		}
	}

	// print out system prompt
	//fmt.Printf("System Prompt:\n%s\n", systemPrompt)

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
		req.Seed = &seed
	}

	// start := time.Now()
	response, err := client.CreateChatCompletion(context.Background(), req)
	// elapsed := time.Since(start)
	// fmt.Printf("Total Execution Time: %s\n", elapsed)

	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}