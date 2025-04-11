package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mitas/puremd-go/pkg/client"
	"github.com/spf13/cobra"
)

var (
	apiKey     string
	timeout    time.Duration
	model      string
	outputFile string
	schemaFile string
)

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "puremd",
		Short: "PureMD API CLI tool",
		Long:  "Command-line interface for interacting with the PureMD API",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Check for API key
			if apiKey == "" {
				apiKey = os.Getenv("PUREMD_API_KEY")
				if apiKey == "" && cmd.Name() != "help" && cmd.Name() != "version" {
					fmt.Println("Error: API key is required. Set it with --key flag or PUREMD_API_KEY environment variable")
					os.Exit(1)
				}
			}
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&apiKey, "key", "", "PureMD API key (can also be set via PUREMD_API_KEY env var)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Request timeout")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Output file (default is stdout)")

	// Fetch command
	fetchCmd := &cobra.Command{
		Use:   "fetch URL",
		Short: "Fetch web content",
		Long:  "Retrieves the content of a given URL in markdown format",
		Args:  cobra.ExactArgs(1),
		Run:   fetchContent,
	}
	rootCmd.AddCommand(fetchCmd)

	// Extract command
	extractCmd := &cobra.Command{
		Use:   "extract URL",
		Short: "Extract data from web content",
		Long:  "Fetches a URL and extracts data based on the prompt",
		Args:  cobra.ExactArgs(1),
		Run:   extractData,
	}
	extractCmd.Flags().StringVarP(&model, "model", "m", string(client.ModelLlama31), "AI model to use")
	extractCmd.Flags().StringVarP(&schemaFile, "schema", "s", "", "JSON schema file")
	rootCmd.AddCommand(extractCmd)

	// Search command
	searchCmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search the web",
		Long:  "Searches the web for a given query and returns results in markdown",
		Args:  cobra.ExactArgs(1),
		Run:   searchWeb,
	}
	rootCmd.AddCommand(searchCmd)

	// Search and extract command
	searchExtractCmd := &cobra.Command{
		Use:   "search-extract QUERY",
		Short: "Search and extract data",
		Long:  "Searches the web and extracts data based on the prompt",
		Args:  cobra.ExactArgs(1),
		Run:   searchAndExtract,
	}
	searchExtractCmd.Flags().StringVarP(&model, "model", "m", string(client.ModelLlama31), "AI model to use")
	searchExtractCmd.Flags().StringVarP(&schemaFile, "schema", "s", "", "JSON schema file")
	rootCmd.AddCommand(searchExtractCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("PureMD CLI v1.0.0")
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createClient() *client.Client {
	return client.NewClient(
		apiKey,
		client.WithTimeout(timeout),
	)
}

func writeOutput(data string) error {
	if outputFile == "" {
		fmt.Print(data)
		return nil
	}

	return os.WriteFile(outputFile, []byte(data), 0644)
}

func fetchContent(cmd *cobra.Command, args []string) {
	url := args[0]
	c := createClient()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	content, err := c.FetchWebContent(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeOutput(content); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func extractData(cmd *cobra.Command, args []string) {
	url := args[0]
	
	// Get prompt from stdin if not provided as an argument
	prompt, err := readStdinOrPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading prompt: %v\n", err)
		os.Exit(1)
	}

	// Prepare extract request
	request := &client.ExtractRequest{
		Prompt: prompt,
		Model:  client.ExtractModel(model),
	}

	// Handle schema if provided
	if schemaFile != "" {
		schema, err := os.ReadFile(schemaFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read schema file: %v\n", err)
			os.Exit(1)
		}
		request.Schema = schema
	}

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.ExtractData(ctx, url, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeOutput(string(result)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func searchWeb(cmd *cobra.Command, args []string) {
	query := args[0]
	c := createClient()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	content, err := c.SearchWeb(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeOutput(content); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func searchAndExtract(cmd *cobra.Command, args []string) {
	query := args[0]
	
	// Get prompt from stdin if not provided as an argument
	prompt, err := readStdinOrPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading prompt: %v\n", err)
		os.Exit(1)
	}

	// Prepare search extract request
	request := &client.SearchExtractRequest{
		Prompt: prompt,
		Model:  client.ExtractModel(model),
	}

	// Handle schema if provided
	if schemaFile != "" {
		schema, err := os.ReadFile(schemaFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read schema file: %v\n", err)
			os.Exit(1)
		}
		request.Schema = schema
	}

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.SearchAndExtract(ctx, query, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeOutput(string(result)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}
}

// readStdinOrPrompt reads from stdin or prompts the user for input
func readStdinOrPrompt() (string, error) {
	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		var prompt string
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading from stdin: %w", err)
		}
		prompt = string(data)
		return prompt, nil
	}
	
	// No data in stdin, prompt the user
	fmt.Fprint(os.Stderr, "Enter prompt: ")
	var prompt string
	if _, err := fmt.Scanln(&prompt); err != nil {
		return "", fmt.Errorf("reading user input: %w", err)
	}
	
	return prompt, nil
}