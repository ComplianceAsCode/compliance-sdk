# Compliance SDK API Documentation

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [Rule Types](#rule-types)
- [Input Types](#input-types)
- [Scanner API](#scanner-api)
- [Fetcher API](#fetcher-api)
- [Builder API](#builder-api)
- [Result Types](#result-types)

## Core Interfaces

### Rule Interface

The base interface for all compliance rules:

```go
type Rule interface {
    // Returns a unique identifier for this rule
    Identifier() string
    
    // Returns the rule type (CEL, Rego, etc.)
    Type() RuleType
    
    // Returns the list of inputs needed for evaluation
    Inputs() []Input
    
    // Returns optional rule metadata for compliance reporting
    Metadata() *RuleMetadata
    
    // Returns the rule-specific content (expression, policy, etc.)
    Content() interface{}
}
```

### CelRule Interface

Extends Rule for CEL-specific functionality:

```go
type CelRule interface {
    Rule
    
    // Returns the CEL expression to evaluate
    Expression() string
}
```

### Input Interface

Defines a generic input that a rule needs:

```go
type Input interface {
    // Name to bind this input to in the rule context
    Name() string
    
    // Type of input (kubernetes, file, system, etc.)
    Type() InputType
    
    // Input specification
    Spec() InputSpec
}
```

## Rule Types

### Supported Rule Types

```go
const (
    RuleTypeCEL      RuleType = "cel"      // CEL expressions
    RuleTypeRego     RuleType = "rego"     // OPA Rego policies
    RuleTypeJSONPath RuleType = "jsonpath" // JSONPath expressions
    RuleTypeCustom   RuleType = "custom"   // Custom implementations
)
```

### Rule Metadata

```go
type RuleMetadata struct {
    Name        string                 `json:"name,omitempty"`
    Description string                 `json:"description,omitempty"`
    Extensions  map[string]interface{} `json:"extensions,omitempty"`
}
```

## Input Types

### Supported Input Types

```go
const (
    InputTypeKubernetes InputType = "kubernetes" // K8s resources
    InputTypeFile       InputType = "file"       // File system
    InputTypeSystem     InputType = "system"     // System services
    InputTypeHTTP       InputType = "http"       // HTTP APIs
    InputTypeDatabase   InputType = "database"   // Databases
)
```

### Kubernetes Input

```go
type KubernetesInputSpec interface {
    InputSpec
    ApiGroup() string     // API group (e.g., "apps", "")
    Version() string      // API version (e.g., "v1")
    ResourceType() string // Resource type (e.g., "pods")
    Namespace() string    // Namespace (empty for cluster-scoped)
    Name() string         // Resource name (empty for all)
}
```

### File Input

```go
type FileInputSpec interface {
    InputSpec
    Path() string             // File or directory path
    Format() string           // Expected format (json, yaml, text)
    Recursive() bool          // For directory traversal
    CheckPermissions() bool   // Include file permissions
}
```

### System Input

```go
type SystemInputSpec interface {
    InputSpec
    ServiceName() string // System service name
    Command() string     // Command to execute
    Args() []string      // Command arguments
}
```

### HTTP Input

```go
type HTTPInputSpec interface {
    InputSpec
    URL() string                // Endpoint URL
    Method() string             // HTTP method
    Headers() map[string]string // Request headers
    Body() []byte               // Request body
}
```

## Scanner API

### Scanner

Main scanner implementation:

```go
type Scanner struct {
    resourceFetcher ResourceFetcher
    logger          Logger
}

// Create a new scanner
func NewScanner(resourceFetcher ResourceFetcher, logger Logger) *Scanner

// Execute compliance checks
func (s *Scanner) Scan(ctx context.Context, config ScanConfig) ([]CheckResult, error)
```

### ScanConfig

Configuration for scanning:

```go
type ScanConfig struct {
    Rules              []Rule        `json:"rules"`
    Variables          []CelVariable `json:"variables"`
    ApiResourcePath    string        `json:"apiResourcePath"`
    EnableDebugLogging bool          `json:"enableDebugLogging"`
}
```

### Logger Interface

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

## Fetcher API

### ResourceFetcher Interface

```go
type ResourceFetcher interface {
    // Fetch resources for a rule
    FetchResources(ctx context.Context, rule Rule, variables []CelVariable) (map[string]interface{}, []string, error)
}
```

### InputFetcher Interface

```go
type InputFetcher interface {
    // Retrieve data for specified inputs
    FetchInputs(inputs []Input, variables []CelVariable) (map[string]interface{}, error)
    
    // Check if fetcher supports an input type
    SupportsInputType(inputType InputType) bool
}
```

### CompositeFetcher

Combines multiple fetchers:

```go
type CompositeFetcher struct {
    // ...
}

// Register a custom fetcher for a specific input type
func (c *CompositeFetcher) RegisterCustomFetcher(inputType InputType, fetcher InputFetcher)

// Get all supported input types
func (c *CompositeFetcher) GetSupportedInputTypes() []InputType
```

### KubernetesFetcher

```go
// Create fetcher with live K8s client
func NewKubernetesFetcher(client runtimeclient.Client, clientset kubernetes.Interface) *KubernetesFetcher

// Create fetcher for pre-fetched files
func NewKubernetesFileFetcher(apiResourcePath string) *KubernetesFetcher

// Configure custom resource mappings
func (k *KubernetesFetcher) WithConfig(config *ResourceMappingConfig) *KubernetesFetcher
```

### FilesystemFetcher

```go
// Create filesystem fetcher with optional base path
func NewFilesystemFetcher(basePath string) *FilesystemFetcher
```

## Builder API

### RuleBuilder

Fluent API for building rules:

```go
// Create a new builder
func NewRuleBuilder(id string, ruleType RuleType) *RuleBuilder

// Add inputs
func (b *RuleBuilder) WithInput(input Input) *RuleBuilder
func (b *RuleBuilder) WithKubernetesInput(name, group, version, resourceType, namespace, resourceName string) *RuleBuilder
func (b *RuleBuilder) WithFileInput(name, path, format string, recursive, checkPermissions bool) *RuleBuilder
func (b *RuleBuilder) WithSystemInput(name, service, command string, args []string) *RuleBuilder
func (b *RuleBuilder) WithHTTPInput(name, url, method string, headers map[string]string, body []byte) *RuleBuilder

// Set rule content
func (b *RuleBuilder) SetCelExpression(expression string) *RuleBuilder
// Future: SetRegoPolicy, SetJSONPathExpression, SetCustomContent methods

// Add metadata
func (b *RuleBuilder) WithMetadata(metadata *RuleMetadata) *RuleBuilder
func (b *RuleBuilder) WithName(name string) *RuleBuilder
func (b *RuleBuilder) WithDescription(description string) *RuleBuilder
func (b *RuleBuilder) WithExtension(key string, value interface{}) *RuleBuilder

// Build the rule
func (b *RuleBuilder) Build() (Rule, error)
func (b *RuleBuilder) BuildCelRule() (CelRule, error)
```

## Result Types

### CheckResult

Result of a compliance check:

```go
type CheckResult struct {
    ID           string              `json:"id"`
    Status       CheckResultStatus   `json:"status"`
    Metadata     CheckResultMetadata `json:"metadata"`
    Warnings     []string            `json:"warnings"`
    ErrorMessage string              `json:"errorMessage"`
}
```

### CheckResultStatus

Possible check statuses:

```go
const (
    CheckResultPass          CheckResultStatus = "PASS"
    CheckResultFail          CheckResultStatus = "FAIL"
    CheckResultError         CheckResultStatus = "ERROR"
    CheckResultNotApplicable CheckResultStatus = "NOT-APPLICABLE"
)
```

### ScanResult (deprecated)

Legacy result type (use CheckResult):

```go
type ScanResult struct {
    RuleID  string                 `json:"ruleId"`
    Status  ScanStatus             `json:"status"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details"`
}
```

## Helper Functions

### Constructor Functions

```go
// Create CEL rules
func NewCelRule(id, expression string, inputs []Input) CelRule
func NewCelRuleWithMetadata(id, expression string, inputs []Input, metadata *RuleMetadata) CelRule

// Future: Constructor functions for Rego, JSONPath, and Custom rule types

// Create inputs
func NewKubernetesInput(name, group, version, resourceType, namespace, resourceName string) Input
func NewFileInput(name, path, format string, recursive bool, checkPermissions bool) Input
func NewSystemInput(name, service, command string, args []string) Input
func NewHTTPInput(name, url, method string, headers map[string]string, body []byte) Input
```

### Utility Functions

```go
// Save scan results to JSON file
func SaveResults(filePath string, results []CheckResult) error

// Derive resource path for Kubernetes resources
func DeriveResourcePath(gvr schema.GroupVersionResource, namespace string) string
```
