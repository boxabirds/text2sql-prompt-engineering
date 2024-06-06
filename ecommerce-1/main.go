package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
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
const MAX_RETRIES = 2

const (
	boldRed   = "\033[1;31m"
	boldGreen = "\033[1;32m"
	reset     = "\033[0m"
)

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

func rows2Json(rows *sql.Rows) (string, error) {
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

	// Iterate over rows
	for rows.Next() {
		rowMap := make(map[string]interface{})
		if err := rows.Scan(scanArgs...); err != nil {
			return "", err
		}

		for i, col := range cols {
			// Ensure the values are properly formatted for JSON
			var value interface{}
			byteVal, ok := colVals[i].([]byte)
			if ok {
				value = string(byteVal)
			} else {
				value = colVals[i]
			}
			rowMap[col] = value
		}

		allRows = append(allRows, rowMap)
	}

	// Handle any errors from iterating over rows
	if err := rows.Err(); err != nil {
		return "", err
	}

	// Convert allRows to JSON
	jsonData, err := json.Marshal(allRows)
	if err != nil {
		return "", err
	}

	//return string(jsonData), nil
	var compactedJSON bytes.Buffer
	err = json.Compact(&compactedJSON, jsonData)
	if err != nil {
		return "", err
	}

	return compactedJSON.String(), nil

}

func stripNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

func main() {
	// deal with command line flags first
	var modelsArg string
	var models []string
	flag.StringVar(&modelsArg, "models", "", "List of models separated by commas")

	baseURL := flag.String("base-url", "", "Base URL for the API server")
	maxTokens := flag.Int("max-tokens", 200, "Maximum number of tokens in the summary")
	var seed int
	flag.IntVar(&seed, "seed", NO_SEED, "Seed for deterministic results (optional)")
	flag.Parse()

	models = strings.Split(modelsArg, ",")
	for i, model := range models {
		models[i] = strings.TrimSpace(model)
	}

	// print out the list of models the user specified
	if len(models) > 0 {
		fmt.Printf("Models to use: %s\n", strings.Join(models, ", "))
	} else {
		fmt.Printf("Using default model: %s\n", DEFAULT_OLLAMA_MODEL)
		models = []string{DEFAULT_OLLAMA_MODEL}
	}

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
	for _, model := range models {
		fmt.Printf("\n\n=======================================\n")
		fmt.Printf("Using model: '%s'\n", model)
		client := createOpenAiClient(*baseURL, &model)

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
					log.Printf("Error predicting SQL for query '%s': %v\n", item.Query, err)
					continue
				}
				// SQL statements are often multi-line but work on a single line so for readability we compress it to a single line
				predictedQuery = stripNewlines(predictedQuery)

				// Execute the SQL query
				rows, err := db.Query(predictedQuery)

				// SQL query failed so let's regenerate the query
				// taking into account this and previous errors
				// by including them in the message sent to the LLM, and try again.
				if err != nil {
					log.Printf("! Error executing query '%s' (%s) generating a new query", predictedQuery, err.Error())
					failedAttempts = append(failedAttempts, FailedSqlQueryAttempt{
						// Compress the sql query to a single line
						SqlQuery:     predictedQuery,
						ErrorMessage: stripNewlines(err.Error()),
					})

					// generating the query was successful, so let's compare against ground truth
				} else {
					successfulSqlQuery = true
					fmt.Printf("- Ground Truth Query: '%s'\n", item.SQL)
					fmt.Printf("- Generated Query:    '%s'\n", predictedQuery)

					jsonRows, _ := rows2Json(rows)

					fmt.Printf("- Ground Truth Result:%s\n", item.Result)
					fmt.Printf("- SQL Result:         %s\n", jsonRows)

					if item.Result == jsonRows {
						fmt.Printf("- And they are the %ssame%s\n\n", boldGreen, reset)
					} else {
						fmt.Printf("- And they are %sdifferent%s\n\n", boldRed, reset)
					}

				}
			}

			if err != nil {
				log.Printf("Failed to execute a valid query after %d attempts for query '%s'.", MAX_RETRIES+1, item.Query)
			}
		}
	}
}

func predictSqlQueryFromNaturalLanguageQuery(client *openai.Client, model string, maxTokens *int, systemPrompt string, query *string, seed int, failedAttempts []FailedSqlQueryAttempt) (string, error) {
	// Modify the system prompt to include the history of failed attempts
	//fmt.Printf("- Query: '%s'\n", *query)
	if len(failedAttempts) > 0 {
		systemPrompt += "\nTake into account the following past failed attempts at generating a SQL query when creating the query to avoid the same mistakes:\n"
		for _, attempt := range failedAttempts {
			systemPrompt += fmt.Sprintf("Generated failed sql query: '%s'; Error message explaining why it failed: '%s'\n", strings.ReplaceAll(attempt.SqlQuery, "\n", " "), strings.ReplaceAll(attempt.ErrorMessage, "\n", " "))
		}
	}

	// print out system prompt
	//fmt.Printf("- System Prompt:\n--------\n%s\n--------\n", strings.ReplaceAll(systemPrompt, "\n", " "))

	req := openai.ChatCompletionRequest{
		Model:       model,
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

	start := time.Now()
	response, err := client.CreateChatCompletion(context.Background(), req)
	elapsed := time.Since(start)
	fmt.Printf("- Query generation execution time: %s\n", elapsed)

	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}
