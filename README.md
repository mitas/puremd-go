# PureMD Go Client

A Go client and CLI for interacting with the [pure.md](https://pure.md) API.

## Features

- Simple, idiomatic Go API client
- Command-line interface for all API operations
- Support for all pure.md endpoints:
  - Fetch web content
  - Extract data from web pages
  - Search the web
  - Search and extract data

## Installation

```bash
go install github.com/mitas/puremd-go/cmd/puremd@latest
```

Or build from source:

```bash
git clone https://github.com/mitas/puremd-go.git
cd puremd-go
go build -o puremd ./cmd/puremd
```

## CLI Usage

Set your API key:

```bash
export PUREMD_API_KEY="your-api-key"
```

### Fetch web content

```bash
puremd fetch https://example.com
```

### Extract data from a webpage

```bash
# Extract with prompt from interactive input
puremd extract https://example.com

# Extract with prompt from pipe
echo "Extract the main heading" | puremd extract https://example.com

# Extract with schema from file
puremd extract --schema schema.json https://example.com
```

### Search the web

```bash
puremd search "golang best practices"
```

### Search and extract data

```bash
# Search and extract with prompt from interactive input
puremd search-extract "golang best practices"

# Search with schema from file
puremd search-extract --schema schema.json "golang best practices"
```

### Other options

```bash
# Output to file
puremd fetch --output results.md https://example.com

# Set custom timeout
puremd fetch --timeout 60s https://example.com

# Specify model for extraction
puremd extract --model meta/llama-3.3-70b https://example.com
```

## Go API Usage

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mitas/puremd-go/pkg/client"
)

func main() {
	// Create client
	c := client.NewClient(
		os.Getenv("PUREMD_API_KEY"),
		client.WithTimeout(30*time.Second),
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch web content
	content, err := c.FetchWebContent(ctx, "https://example.com")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(content)

	// Extract data
	request := &client.ExtractRequest{
		Prompt: "Extract the main heading",
		Model:  client.ModelLlama31,
	}
	result, err := c.ExtractData(ctx, "https://example.com", request)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(string(result))
}
```

## License

MIT