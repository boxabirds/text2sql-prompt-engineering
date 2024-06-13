package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const GroundTruthMdFile = "ground-truth.md"
const NoSeed = -1
const (
	boldRed   = "\033[1;31m"
	boldGreen = "\033[1;32m"
	reset     = "\033[0m"
)

// Determine what kind of match we have between the ground truth and the generated SQL query.

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

// func createOpenAiClient(baseUrl string, model *string) *openai.Client {
// 	var apiKey string
// 	if baseUrl == "" {
// 		apiKey = os.Getenv("OPENAI_API_KEY")
// 		if apiKey == "" {
// 			log.Fatal("OPENAI_API_KEY not found in environment")
// 		}
// 		if *model == "" {
// 			*model = DEFAULT_OPENAI_MODEL
// 		}
// 		fmt.Printf("Using default OpenAI API server\n")
// 	} else {
// 		//fmt.Printf("=== NOTE AS OF 28 May 2024 Ollama does not appear to use the seed to make output deterministic.===")
// 		apiKey = OLLAMA_API_KEY
// 		if *model == "" {
// 			*model = DEFAULT_OLLAMA_MODEL
// 		}
// 		fmt.Printf("Using custom API server at: %s\n", baseUrl)
// 		fmt.Printf("API Key set to Ollama\n")
// 	}
// 	fmt.Printf("Model being used: %s\n", *model)

// 	config := openai.DefaultConfig(apiKey)

// 	// have to check twice because the config that's created and depends on it
// 	// and yet needs to be changed again
// 	if baseUrl != "" {
// 		config.BaseURL = baseUrl
// 	}

// 	return openai.NewClientWithConfig(config)
// }

// Take some Row rules and conver them to JSON format for easy parsing
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
	baseURL := flag.String("base-url", "", "Base URL for the API server")
	maxTokens := flag.Int("max-tokens", 200, "Maximum number of tokens in the summary")
	var seed int
	flag.IntVar(&seed, "seed", NoSeed, "Seed for deterministic (in theory) results (optional)")
	flag.Parse()

	llmClients := initialiseLLMClients(*baseURL)
	// print out the initialisers: name and model
	for _, llm := range llmClients {
		fmt.Printf("LLM: %s\n", llm.Name)
		fmt.Printf("Model: %s\n", llm.Model)
	}

	// for our evaluation we use one of the SOTA models
	LLMevaluator := getLLMClient("Ollama/OpenAI", "phi3:medium", llmClients)
	fmt.Printf("Evaluator selected %s %s\n", LLMevaluator.Name, LLMevaluator.Model)

	// ensure our db exists and has the content we want to test against
	db, err := initialiseDb("ecommerce-autogen.db")
	if err != nil {
		fmt.Println(err)
	}

	// ensure we have our ground truth MD file in a CSV file for easy processing
	groundTruthCsvFile, err := convertMdWithSingleTableToCsv(GroundTruthMdFile)
	if err == nil {
		log.Fatalf("Failed to convert MD to CSV: %v", err)
		// load the ground truth
	}
	groundTruth, err := loadGroundTruthCsv(groundTruthCsvFile)
	if err != nil {
		log.Fatalf("Failed to load CSV: %v", err)
	}
	//fmt.Printf("Loaded %d ground truth items\n", len(groundTruth))

	// do the AI stuff to predict the SQL query from natural language

	for _, llmClient := range llmClients {
		fmt.Printf("\n\n=======================================\n")
		fmt.Printf("Using model: %s %s\n", llmClient.Name, llmClient.Model)

		systemPrompt := SqlGeneratorApiSystemPrompt + strings.Join(TABLES, "\n")

		for _, item := range groundTruth {
			var failedAttempts []FailedSqlQueryAttempt
			var predictedSqlQuery string
			var err error
			fmt.Printf("\n==== %s: %s\n", llmClient.Name, llmClient.Model)

			successfulSqlQuery := false

			for len(failedAttempts) <= MaxSqlGenerationFaultRetries && !successfulSqlQuery {
				// predict the SQL query from the natural language query
				// print out the natural query
				fmt.Printf("Query: %s\n", item.Query)
				predictedSqlQuery, err = predictSqlQueryFromNaturalLanguageQuery(llmClient.Instance, maxTokens, systemPrompt, &item.Query, seed, failedAttempts)
				if err != nil {
					log.Printf("Error predicting SQL for query '%s': %v\n", item.Query, err)
					break
				}
				// SQL statements are often multi-line but work on a single line so for readability we compress it to a single line
				predictedSqlQuery = stripNewlines(predictedSqlQuery)

				// Execute the SQL query
				rows, err := db.Query(predictedSqlQuery)

				// SQL query failed so let's regenerate the query
				// taking into account this and previous errors
				// by including them in the message sent to the LLM, and try again.
				if err != nil {
					log.Printf("! Error executing query '%s' (%s) generating a new query", predictedSqlQuery, err.Error())
					failedAttempts = append(failedAttempts, FailedSqlQueryAttempt{
						// Compress the sql query to a single line
						SqlQuery:     predictedSqlQuery,
						ErrorMessage: stripNewlines(err.Error()),
					})

					// generating the query was successful, so let's compare against ground truth
				} else {
					successfulSqlQuery = true
					fmt.Printf("- Ground Truth Query: '%s'\n", item.SQL)
					fmt.Printf("- Generated Query:    '%s'\n", predictedSqlQuery)

					sqlQueryComparison, err := compareSqlQueries(item.SQL, predictedSqlQuery, LLMevaluator, maxTokens, seed)
					if err != nil {
						log.Printf("Error comparing SQL queries: %v", err)
					}
					fmt.Printf("- SQL Query Comparison result: %s\n", sqlQueryComparison)
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

			if !successfulSqlQuery {
				log.Printf("Failed to execute a valid query after %d attempts for query '%s'.", MaxSqlGenerationFaultRetries+1, item.Query)
			}
		}
	}
}
