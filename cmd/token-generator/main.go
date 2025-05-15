package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	//"github.com/ceesaxp/cocktail-bot/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	// Default token length in bytes (resulting in longer base64 string)
	defaultTokenLength = 24
	// Default output file
	defaultTokensFile = "api_tokens.yaml"
)

type tokensFile struct {
	AuthTokens []string `yaml:"auth_tokens"`
}

func main() {
	// Define command-line flags
	numTokens := flag.Int("count", 1, "Number of tokens to generate")
	tokenLength := flag.Int("length", defaultTokenLength, "Length of generated tokens in bytes (before encoding)")
	outputFile := flag.String("output", defaultTokensFile, "Output file for tokens")
	appendTokens := flag.Bool("append", false, "Append to existing tokens file instead of overwriting")
	displayOnly := flag.Bool("display-only", false, "Only display tokens, don't write to file")
	force := flag.Bool("force", false, "Force overwrite of existing file without confirmation")

	flag.Parse()

	fmt.Println("Cocktail Bot API Token Generator")
	fmt.Println("===============================")

	if *numTokens <= 0 {
		log.Fatal("Number of tokens must be greater than 0")
	}

	// Generate tokens
	tokens, err := generateTokens(*numTokens, *tokenLength)
	if err != nil {
		log.Fatalf("Error generating tokens: %v", err)
	}

	// Display tokens
	fmt.Println("\nGenerated Tokens:")
	for i, token := range tokens {
		fmt.Printf("%d: %s\n", i+1, token)
	}

	// If display-only mode, exit here
	if *displayOnly {
		fmt.Println("\nTokens displayed only, not saved to file.")
		return
	}

	// Check if file exists
	var existingTokens []string
	if *appendTokens {
		existingTokens, err = readExistingTokens(*outputFile)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("Error reading existing tokens: %v", err)
		}
	} else if !*force {
		if _, err := os.Stat(*outputFile); err == nil {
			fmt.Printf("\nWarning: File %s already exists. Overwrite? (y/n): ", *outputFile)
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				fmt.Println("Error reading response:", err)
				response = "n" // Default to 'no' if error
			}
			if !strings.HasPrefix(strings.ToLower(response), "y") {
				fmt.Println("Operation cancelled.")
				return
			}
		}
	}

	// Combine existing and new tokens if appending
	outputTokens := tokens
	if *appendTokens && len(existingTokens) > 0 {
		// Create a map to check for duplicates
		tokenMap := make(map[string]bool)
		for _, t := range existingTokens {
			tokenMap[t] = true
		}

		// Add new tokens, checking for duplicates
		for _, t := range tokens {
			if !tokenMap[t] {
				existingTokens = append(existingTokens, t)
			} else {
				fmt.Printf("Warning: Token %s already exists in file, skipping.\n", t)
			}
		}

		outputTokens = existingTokens
	}

	// Write tokens to file
	err = writeTokensToFile(*outputFile, outputTokens)
	if err != nil {
		log.Fatalf("Error writing tokens to file: %v", err)
	}

	fmt.Printf("\nSuccessfully wrote %d tokens to %s\n", len(outputTokens), *outputFile)
	fmt.Println("\nTo use these tokens in your REST API:")
	fmt.Println("1. Set API_ENABLED=true in your configuration")
	fmt.Println("2. Include the token in API requests with:")
	fmt.Println("   Authorization: Bearer <token>")

	// Provide configuration tip
	fmt.Printf("\nConfiguration tip:\n")
	fmt.Printf("Make sure your config.yaml has API settings:\n\n")
	fmt.Printf("api:\n")
	fmt.Printf("  enabled: true\n")
	fmt.Printf("  port: 8080\n")
	fmt.Printf("  tokens_file: \"%s\"\n", *outputFile)
	fmt.Printf("  rate_limit_per_min: 30\n")
	fmt.Printf("  rate_limit_per_hour: 300\n")
}

// generateTokens generates the specified number of random tokens
func generateTokens(count, length int) ([]string, error) {
	tokens := make([]string, count)

	for i := 0; i < count; i++ {
		// Generate random bytes
		randomBytes := make([]byte, length)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}

		// Convert to base64 and remove any non-alphanumeric characters
		token := base64.RawURLEncoding.EncodeToString(randomBytes)
		tokens[i] = token
	}

	return tokens, nil
}

// readExistingTokens reads tokens from an existing file
func readExistingTokens(filename string) ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	// Read and parse file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tokens tokensFile
	err = yaml.Unmarshal(data, &tokens)
	if err != nil {
		return nil, fmt.Errorf("error parsing tokens file: %w", err)
	}

	return tokens.AuthTokens, nil
}

// writeTokensToFile writes tokens to a YAML file
func writeTokensToFile(filename string, tokens []string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create tokens structure
	tokenData := tokensFile{
		AuthTokens: tokens,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("error marshaling tokens: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0600) // Restrictive permissions for security
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}
