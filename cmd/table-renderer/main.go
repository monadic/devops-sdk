// table-renderer - CLI tool to render JSON data as ASCII tables
// Usage: echo '{"headers":["Name","Age"],"rows":[["Alice","30"],["Bob","25"]]}' | table-renderer
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	sdk "github.com/monadic/devops-sdk"
)

type TableInput struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
	Style   string     `json:"style"` // "default", "simple", "double", "none"
}

func main() {
	// Read JSON from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	var input TableInput
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Create table
	table := sdk.NewTableWriter(input.Headers)

	// Set border style
	switch input.Style {
	case "simple":
		table.SetBorderStyle(sdk.SimpleBorder)
	case "double":
		table.SetBorderStyle(sdk.DoubleBorder)
	case "none":
		table.SetBorderStyle(sdk.NoBorder)
	default:
		table.SetBorderStyle(sdk.DefaultBorder)
	}

	// Add rows
	for _, row := range input.Rows {
		table.AddRow(row)
	}

	// Render and output
	fmt.Println(table.Render())
}
