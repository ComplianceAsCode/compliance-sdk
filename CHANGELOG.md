# Changelog

All notable changes to the Compliance SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-20

### Initial Release

#### Features
- **CEL Rule Engine**: Full support for Common Expression Language (CEL) for complex logical expressions
- **Extensible Architecture**: Generic `Rule` interface designed to support multiple rule engines in the future
- **Multiple Input Sources**:
  - Kubernetes resources with discovery and caching
  - File system resources with format parsing (JSON, YAML, text)
  - System services and processes
  - HTTP API endpoints
- **Resource Fetching**:
  - Composite fetcher for unified resource retrieval
  - Concurrent resource fetching for performance
  - Support for pre-fetched resources (offline scanning)
- **Builder Pattern**: Fluent API for constructing rules programmatically
- **Rich Metadata**: Support for rule metadata and extensions for compliance reporting
- **Comprehensive Testing**: Unit tests with mock implementations
- **Documentation**: Complete API documentation and usage examples

#### Architecture
- Generic `Rule` interface as the foundation for all rule types
- `CelRule` interface extending `Rule` for CEL-specific functionality
- Pluggable `InputFetcher` system for different resource types
- Clean separation between rule definition, resource fetching, and evaluation

#### Future Rule Types (Planned)
The architecture supports these rule types, with implementations planned for future releases:
- **Rego**: For OPA (Open Policy Agent) policy language
- **JSONPath**: For simple path-based validations  
- **Custom**: For user-defined rule implementations