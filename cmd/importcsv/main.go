package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Ceesaxp/cocktail-bot/internal/utils"
)

func main() {
	inputFile := flag.String("input", "", "Input CSV file with emails")
	outputFile := flag.String("output", "./data/users.csv", "Output CSV file for bot database")
	column := flag.Int("column", 1, "Column number containing emails (1-based)")
	hasHeader := flag.Bool("header", true, "Input file has a header row")

	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Error: Input file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Open input file
	input, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer input.Close()

	// Create output file
	output, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	// Create CSV readers and writers
	reader := csv.NewReader(input)
	writer := csv.NewWriter(output)
	defer writer.Flush()

	// Write header to output
	if err := writer.Write([]string{"ID", "Email", "Date Added", "Already Consumed"}); err != nil {
		fmt.Printf("Error writing header: %v\n", err)
		os.Exit(1)
	}

	// Process input file
	rowNum := 0
	emailsAdded := 0
	invalidEmails := 0
	duplicateEmails := make(map[string]bool)

	for {
		record, err := reader.Read()
		if err != nil {
			break // End of file
		}

		rowNum++

		// Skip header if present
		if rowNum == 1 && *hasHeader {
			continue
		}

		// Check if column index is valid
		if *column < 1 || *column > len(record) {
			fmt.Printf("Error: Column %d is out of range for row %d\n", *column, rowNum)
			continue
		}

		// Get email from specified column
		email := strings.TrimSpace(record[*column-1])
		email = utils.NormalizeEmail(email)

		// Skip if email is empty
		if email == "" {
			continue
		}

		// Validate email
		if !utils.IsValidEmail(email) {
			fmt.Printf("Invalid email at row %d: %s\n", rowNum, email)
			invalidEmails++
			continue
		}

		// Check for duplicates
		if duplicateEmails[email] {
			fmt.Printf("Duplicate email at row %d: %s\n", rowNum, email)
			continue
		}
		duplicateEmails[email] = true

		// Write to output
		if err := writer.Write([]string{
			fmt.Sprintf("%d", rowNum),
			email,
			time.Now().Format(time.RFC3339),
			"", // Empty Already Consumed field
		}); err != nil {
			fmt.Printf("Error writing row: %v\n", err)
			os.Exit(1)
		}

		emailsAdded++
	}

	fmt.Printf("Import completed: %d emails added, %d invalid emails\n", emailsAdded, invalidEmails)
}
