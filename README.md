# Compliance SDK

[![Go Version](https://img.shields.io/badge/go-1.24.0-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A flexible and extensible compliance scanning SDK for Go that supports multiple rule engines and input sources. The SDK is designed to evaluate compliance rules against various resource types including Kubernetes resources, files, system services, and HTTP endpoints.

## Features

- **Extensible Rule Engine Architecture**: Currently supports CEL (Common Expression Language) with a design ready for future rule engines (Rego, JSONPath, custom)
- **Multiple Input Sources**: 
  - Kubernetes resources
  - File system resources (with format parsing support)
  - System services and processes (planned)
  - HTTP API endpoints (planned)
  - Database queries (planned)
- **Flexible Architecture**: Clean separation between rule definitions, resource fetching, and evaluation
- **Builder Pattern**: Fluent API for constructing rules programmatically
- **Rich Metadata Support**: Attach metadata and extensions to rules for compliance reporting
- **Concurrent Resource Fetching**: Efficient parallel fetching of resources
- **File-based Scanning**: Support for pre-fetched resources for offline scanning

## Installation

```bash
go get github.com/ComplianceAsCode/compliance-sdk
```

## Quick Start

### 1. Creating a CEL Rule

```go
import (
    "github.com/ComplianceAsCode/compliance-sdk/pkg/scanner"
)

// Create a rule using the builder pattern
rule, err := scanner.NewRuleBuilder("pod-security-check", scanner.RuleTypeCEL).
    WithKubernetesInput("pods", "", "v1", "pods", "kube-system", "").
    SetCelExpression(`pods.items.all(pod, 
        pod.spec.securityContext.runAsNonRoot == true && 
        pod.spec.containers.all(c, c.securityContext.allowPrivilegeEscalation == false)
    )`).
    WithName("Pod Security Check").
    WithDescription("Ensures all pods in kube-system follow security best practices").
    WithExtension("severity", "high").
    BuildCelRule()
```

### 2. Setting up a Scanner

```go
import (
    "github.com/ComplianceAsCode/compliance-sdk/pkg/scanner"
    "github.com/ComplianceAsCode/compliance-sdk/pkg/fetchers"
)

// Create a composite fetcher that supports multiple input types
fetcher := fetchers.NewCompositeFetcher()

// Create a scanner instance
scannerInstance := scanner.NewScanner(fetcher, nil)

// Configure and run the scan
config := scanner.ScanConfig{
    Rules: []scanner.Rule{rule},
    Variables: []scanner.CelVariable{
        // Add any variables needed for rule evaluation
    },
}

results, err := scannerInstance.Scan(context.Background(), config)
```

### 3. Processing Results

```go
for _, result := range results {
    fmt.Printf("Rule: %s\n", result.ID)
    fmt.Printf("Status: %s\n", result.Status)
    
    if result.Status == scanner.CheckResultFail {
        fmt.Printf("Failed: %s\n", result.ErrorMessage)
    }
}
```

## Core Components

### Rules

The SDK uses a generic `Rule` interface that can be implemented by different rule types:

```go
type Rule interface {
    Identifier() string
    Type() RuleType
    Inputs() []Input
    Metadata() *RuleMetadata
    Content() interface{}
}
```

Currently supported rule types:
- **CEL (Common Expression Language)**: For complex logical expressions
- **Rego** (planned): For OPA policy language support
- **JSONPath** (planned): For simple path-based validations
- **Custom** (planned): For custom rule implementations

### Inputs

Inputs define what resources a rule needs for evaluation:

```go
// Kubernetes resources
input := scanner.NewKubernetesInput("pods", "", "v1", "pods", "default", "")

// File system resources
input := scanner.NewFileInput("config", "/etc/config.yaml", "yaml", false, true)

// System services
input := scanner.NewSystemInput("nginx", "nginx", "", []string{})

// HTTP endpoints
input := scanner.NewHTTPInput("api", "https://api.example.com/health", "GET", nil, nil)
```

### Fetchers

Fetchers retrieve resources based on input specifications:

- **KubernetesFetcher**: Fetches Kubernetes resources using client-go
- **FilesystemFetcher**: Reads and parses files (JSON, YAML, text)
- **CompositeFetcher**: Combines multiple fetchers for unified resource retrieval

### Variables

Variables can be passed to rules for dynamic evaluation:

```go
type CelVariable interface {
    Name() string
    Namespace() string
    Value() string
    GroupVersionKind() schema.GroupVersionKind
}
```

## Advanced Usage

### Building Complex Rules

```go
rule, err := scanner.NewRuleBuilder("complex-check", scanner.RuleTypeCEL).
    // Add multiple inputs
    WithKubernetesInput("pods", "", "v1", "pods", "", "").
    WithKubernetesInput("services", "", "v1", "services", "", "").
    WithFileInput("config", "/etc/app/config.yaml", "yaml", false, false).
    
    // Set CEL expression using all inputs
    SetCelExpression(`
        pods.items.all(pod, 
            services.items.exists(svc, 
                svc.spec.selector.all(k, v, pod.metadata.labels[k] == v)
            )
        ) && config.security.enabled == true
    `).
    
    // Add metadata
    WithName("Service Coverage Check").
    WithDescription("Ensures all pods have corresponding services").
    WithExtension("category", "networking").
    WithExtension("severity", "medium").
    
    BuildCelRule()
```

### Custom Fetcher Implementation

```go
type CustomFetcher struct {
    // your fields
}

func (c *CustomFetcher) FetchInputs(inputs []scanner.Input, variables []scanner.CelVariable) (map[string]interface{}, error) {
    // Implementation
}

func (c *CustomFetcher) SupportsInputType(inputType scanner.InputType) bool {
    // Return true for supported input types
}
```

### File-Based Scanning

For offline or pre-fetched resource scanning:

```go
config := scanner.ScanConfig{
    Rules:           rules,
    ApiResourcePath: "/path/to/fetched/resources",
}
```

## Rule Expression Examples

### CEL Expressions

```go
// Check if all pods have resource limits
"pods.items.all(pod, pod.spec.containers.all(c, has(c.resources.limits)))"

// Verify namespace labels
"namespaces.items.all(ns, has(ns.metadata.labels.environment))"

// Complex multi-resource validation
"deployments.items.all(d, d.spec.replicas >= 2) && services.items.size() > 0"
```

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│     Rules       │     │   Scanner    │     │  Fetchers   │
├─────────────────┤     ├──────────────┤     ├─────────────┤
│ • CEL           │────▶│ • Compile    │────▶│ • K8s       │
│ • Rego (future) │     │ • Evaluate   │     │ • Files     │
│ • Custom        │     │ • Report     │     │ • HTTP      │
└─────────────────┘     └──────────────┘     └─────────────┘
```

## Development

### Running Tests

```bash
make test
```

### Running Tests with Coverage

```bash
make test-coverage
```

### Code Quality

```bash
make fmt  # Format code
make lint # Run linter
```

## Future Enhancements

- [ ] Rego rule engine support
- [ ] JSONPath rule engine support
- [ ] Database input fetcher
- [ ] Rule validation and testing framework
- [ ] Rule composition and inheritance
- [ ] Result aggregation and reporting
- [ ] Remediation suggestions

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [CEL-Go](https://github.com/google/cel-go) for expression evaluation
- Uses [client-go](https://github.com/kubernetes/client-go) for Kubernetes interactions
