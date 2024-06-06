package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/russross/blackfriday/v2"
)

func convertMdWithSingleTableToCsv(fileName string) (string, error) {
	// Read the Markdown file
	input, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return "", err
	}

	// Parse the Markdown to extract the nodes
	node := blackfriday.New(blackfriday.WithExtensions(blackfriday.Tables)).Parse(input)

	// Prepare to find the first table
	var inTable bool
	var lines [][]string

	// Walk through the nodes
	node.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		switch node.Type {
		case blackfriday.Table:
			if entering {
				inTable = true
			} else {
				inTable = false
				return blackfriday.Terminate
			}
		case blackfriday.TableRow:
			if inTable && entering {
				// Each TableRow node represents a row in the table
				var row []string
				node.Walk(func(n *blackfriday.Node, entering bool) blackfriday.WalkStatus {
					if n.Type == blackfriday.TableCell && entering {
						// Extract and clean the text from each TableCell
						text := getTextFromNode(n)
						row = append(row, strings.TrimSpace(text))
					}
					return blackfriday.GoToNext
				})
				lines = append(lines, row)
			}
		}
		return blackfriday.GoToNext
	})

	// Write CSV output
	if len(lines) > 0 {
		outputFile, err := os.Create(fileName + ".csv")
		if err != nil {
			fmt.Println("Error creating CSV file:", err)
			return "", err
		}
		defer outputFile.Close()

		csvWriter := csv.NewWriter(outputFile)
		err = csvWriter.WriteAll(lines) // Write all lines at once
		if err != nil {
			fmt.Println("Error writing to CSV file:", err)
			return "", err
		}
		csvWriter.Flush()
		fmt.Println("CSV file has been created successfully.")
		return outputFile.Name(), nil
	} else {
		fmt.Println("No table found in the Markdown file.")
		return "", nil
	}
}

// getTextFromNode extracts and concatenates text from a Markdown node recursively
func getTextFromNode(node *blackfriday.Node) string {
	var buffer bytes.Buffer
	if node.FirstChild != nil {
		n := node.FirstChild
		for n != nil {
			buffer.WriteString(getTextFromNode(n))
			n = n.Next
		}
	} else {
		return string(node.Literal)
	}
	return buffer.String()
}
