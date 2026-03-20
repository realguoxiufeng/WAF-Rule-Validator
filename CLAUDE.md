# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoTestWAF is a tool for API and OWASP attack simulation that evaluates web application security solutions (WAFs, API gateways, IPS). It generates malicious requests using encoded payloads placed in various parts of HTTP requests and measures how effectively the security solution blocks them.

## Build and Test Commands

```bash
# Build binary (with version from git tags)
make gotestwaf_bin
# Or directly:
go build -o gotestwaf -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(git describe --tags)" ./cmd/gotestwaf

# Run all tests
go test -count=1 -v ./...

# Run tests for a specific package
go test -count=1 -v ./internal/db/...

# Run integration tests
go test -count=1 -v ./tests/integration/...

# Run linter
golangci-lint -v run ./...

# Format code
go fmt ./...
goimports -local "github.com/wallarm/gotestwaf" -w <files>

# Tidy dependencies
go mod tidy
```

## Running GoTestWAF

```bash
# Basic scan
./gotestwaf --url=http://target-url --noEmailReport

# With gRPC testing
./gotestwaf --url=http://target-url --grpcPort 9000 --noEmailReport

# Using OpenAPI spec for request templates
./gotestwaf --url=http://target-url --openapiFile api.yaml --noEmailReport

# With custom test cases
./gotestwaf --url=http://target-url --testCasesPath ./custom-testcases --noEmailReport
```

## Architecture

### Main Entry Point

`cmd/gotestwaf/main.go` is the main entry point. It handles:
- Flag parsing and config loading
- WAF identification and pre-check
- Scanner initialization and execution
- Report generation and export

### Core Data Flow

1. **Test Case Loading** (`internal/db/load.go`): YAML test cases from `testcases/` are loaded into `db.Case` structures containing payloads, encoders, and placeholders.

2. **Payload Generation** (`internal/scanner/scanner.go`): The scanner generates all combinations of payload × encoder × placeholder. For each combination, it creates a request and sends it to the target.

3. **Request Creation** (`internal/payload/`):
   - **Encoders** (`encoder/`): Transform payloads (Base64, URL, JSUnicode, Plain, XMLEntity)
   - **Placeholders** (`placeholder/`): Insert encoded payloads into request locations (URLPath, URLParam, Header, JSONBody, HTMLForm, gRPC, GraphQL, etc.)

4. **HTTP Clients** (`internal/scanner/clients/`):
   - `gohttp/`: Standard Go HTTP client
   - `chrome/`: Headless Chrome via chromedp
   - `grpc/`: gRPC client
   - `graphql/`: GraphQL client

5. **WAF Detection** (`internal/scanner/waf_detector/`): Identifies specific WAF products by analyzing responses.

6. **Result Tracking** (`internal/db/database.go`): Tracks blocked/bypassed/unresolved/failed tests per test set and case.

7. **Reporting** (`internal/report/`): Generates console, HTML, PDF, and JSON reports.

### Key Packages

- `internal/config/`: CLI configuration and settings
- `internal/db/`: Test case database and statistics
- `internal/payload/`: Payload encoding and placeholder injection
- `internal/scanner/`: Main scanning logic and HTTP clients
- `internal/openapi/`: OpenAPI specification parsing for request templates
- `internal/report/`: Report generation in multiple formats
- `pkg/dnscache/`: DNS caching utility (exported package)
- `pkg/report/`: Report validation and helpers (exported package)

### Key Interfaces

The scanner uses interfaces defined in `internal/scanner/clients/clients.go` to abstract protocol handling:

- **HTTPClient**: `SendPayload(ctx, targetURL, payloadInfo)` and `SendRequest(ctx, req)` - implemented by `gohttp` and `chrome` clients
- **GRPCClient**: `CheckAvailability()`, `IsAvailable()`, `SendPayload()`, `Close()` - gRPC protocol handler
- **GraphQLClient**: Similar pattern for GraphQL endpoints

The `types.Request` and `types.Response` interfaces (`internal/scanner/types/`) abstract request/response handling across different client types.

### Test Case Format

YAML files in `testcases/` define attack vectors:

```yaml
payload:
  - "malicious string 1"
  - "malicious string 2"
encoder:
  - Base64Flat
  - URL
placeholder:
  - URLPath
  - JSONRequest
type: SQL Injection
```

Each file produces `len(payload) × len(encoder) × len(placeholder)` test requests.

### True-Positive vs True-Negative Tests

- Test cases in `testcases/owasp/` and `testcases/owasp-api/` are true-positive tests (malicious payloads that should be blocked)
- Test cases in `testcases/false-pos/` are true-negative tests (benign content that should pass through)
- Test cases in `testcases/community/` are community-contributed attack vectors

### Platform-Specific Code

Signal handling uses build constraints:
- `scanner_signal_handler_unix.go` - Unix SIGUSR1 handling for status updates
- `scanner_signal_handler_windows.go` - Windows stub (no signal support)

### Version Information

Version is set at build time via ldflags: `github.com/wallarm/gotestwaf/internal/version.Version`