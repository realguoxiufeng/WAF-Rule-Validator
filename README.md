# WAF Rule Validator

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/Release-v1.0.0-blue.svg)](https://github.com/realguoxiufeng/WAF-Rule-Validator/releases)

[中文文档](README_CN.md)

A tool for evaluating web application security solutions (WAFs, API gateways, IPS) through API and OWASP attack simulation. It generates malicious requests using encoded payloads placed in various parts of HTTP requests and measures how effectively the security solution blocks them.

## Features

- **Multi-Protocol Support**: REST, GraphQL, gRPC, SOAP, XMLRPC, and more
- **Multiple Encoders**: Base64, URL, JSUnicode, Plain, XML Entity
- **Multiple Injection Points**: URL path, URL parameters, headers, request body, JSON, HTML forms, etc.
- **OpenAPI Integration**: Generate request templates based on OpenAPI specifications
- **WAF Detection**: Automatically identify major WAF products (Akamai, F5, Imperva, ModSecurity, etc.)
- **Multi-Format Reports**: PDF, HTML, JSON, and DOCX evaluation reports

## Quick Start

### Requirements

- Go 1.24 or higher
- Chrome browser (optional, for PDF report generation)

### Build

```bash
# Clone the repository
git clone https://github.com/realguoxiufeng/WAF-Rule-Validator.git
cd WAF-Rule-Validator

# Build binary
make gotestwaf_bin

# Or directly with go build
go build -o gotestwaf ./cmd/gotestwaf
```

### Basic Usage

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

### Docker

```bash
# Pull and run
docker pull wallarm/gotestwaf
docker run --rm --network="host" -v ${PWD}/reports:/app/reports \
    wallarm/gotestwaf --url=<TARGET_URL> --noEmailReport
```

## Test Case Format

Test cases are defined in YAML format:

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

- **payload**: Malicious attack payloads
- **encoder**: Encoding methods to apply to payloads
- **placeholder**: Request locations where payloads are injected
- **type**: Attack type name

Each test case file generates `len(payload) × len(encoder) × len(placeholder)` test requests.

## Test Case Directories

| Directory | Description |
|-----------|-------------|
| `testcases/owasp/` | OWASP Top-10 attack vectors (true-positive tests, should be blocked) |
| `testcases/owasp-api/` | OWASP API security attack vectors |
| `testcases/false-pos/` | True-negative tests (benign content, should pass through) |
| `testcases/community/` | Community-contributed attack vectors |

## Supported Encoders

| Encoder | Description |
|---------|-------------|
| Base64 | Base64 encoding |
| Base64Flat | Base64 encoding without padding |
| URL | URL encoding |
| JSUnicode | JavaScript Unicode encoding |
| Plain | Raw text (no encoding) |
| XML Entity | XML entity encoding |

## Supported Placeholders

| Placeholder | Description |
|-------------|-------------|
| URLPath | URL path |
| URLParam | URL parameters |
| Header | HTTP request headers |
| UserAgent | User-Agent header |
| RequestBody | Request body |
| JSONBody | JSON request body |
| JSONRequest | JSON request |
| HTMLForm | HTML form |
| HTMLMultipartForm | Multipart form |
| SOAPBody | SOAP message body |
| XMLBody | XML request body |
| gRPC | gRPC request |
| GraphQL | GraphQL request |
| RawRequest | Raw HTTP request |

## Configuration Options

```bash
Usage: ./gotestwaf [OPTIONS] --url <URL>

Options:
      --url string              Target URL (required)
      --grpcPort uint16         gRPC port
      --graphqlURL string       GraphQL URL
      --openapiFile string      OpenAPI specification file path
      --testCasesPath string    Test cases directory path (default "testcases")
      --testCase string         Run only specified test case
      --testSet string          Run only specified test set
      --httpClient string       HTTP client type: chrome, gohttp (default "gohttp")
      --workers int             Number of concurrent workers (default 5)
      --blockStatusCodes ints   HTTP status codes for blocked requests (default [403])
      --passStatusCodes ints    HTTP status codes for passed requests (default [200,404])
      --blockRegex string       Regex to identify blocking pages
      --passRegex string        Regex to identify normal pages
      --reportFormat strings    Report format: none, json, html, pdf, docx (default [pdf])
      --reportPath string       Directory to store reports (default "reports")
      --reportName string       Report file name
      --noEmailReport           Save report locally without sending email
      --wafName string          WAF product name (default "generic")
      --skipWAFIdentification   Skip WAF identification
      --version                 Show version information
```

## Development

### Running Tests

```bash
# Run all tests
go test -count=1 -v ./...

# Run tests for specific package
go test -count=1 -v ./internal/db/...

# Run integration tests
go test -count=1 -v ./tests/integration/...
```

### Code Quality

```bash
# Run linter
golangci-lint -v run ./...

# Format code
go fmt ./...
goimports -local "github.com/wallarm/gotestwaf" -w <files>
```

### Project Structure

```
.
├── cmd/gotestwaf/          # Main entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── db/                 # Test case database and statistics
│   ├── payload/            # Payload encoding and placeholder injection
│   │   ├── encoder/        # Encoder implementations
│   │   └── placeholder/    # Placeholder implementations
│   ├── scanner/            # Scanning logic and HTTP clients
│   │   └── clients/        # HTTP/gRPC/GraphQL clients
│   ├── openapi/            # OpenAPI specification parsing
│   └── report/             # Report generation
├── pkg/                    # Exported packages
│   ├── dnscache/           # DNS caching utility
│   └── report/             # Report validation and helpers
├── testcases/              # Test cases
└── tests/integration/      # Integration tests
```

## License

This project is licensed under the [MIT License](LICENSE).

## Acknowledgments

This project is based on [GoTestWAF](https://github.com/wallarm/gotestwaf) developed by Wallarm.