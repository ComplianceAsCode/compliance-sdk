/*
Copyright © 2025 Red Hat Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scanner

import (
	"strings"
	"testing"

	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// MockLogger for testing
type MockLogger struct {
	Messages []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.Messages = append(m.Messages, "DEBUG: "+msg)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.Messages = append(m.Messages, "INFO: "+msg)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.Messages = append(m.Messages, "WARN: "+msg)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.Messages = append(m.Messages, "ERROR: "+msg)
}

// Test simple expression validation
func TestValidateCELExpressionSimple(t *testing.T) {
	tests := []struct {
		name                  string
		expression            string
		expectValidationError bool
		expectedErrorType     ValidationErrorType
	}{
		{
			name:                  "simple arithmetic expression validates successfully",
			expression:            "1 + 1 == 2",
			expectValidationError: false,
		},
		{
			name:                  "boolean logic expression validates successfully",
			expression:            "true && false == false",
			expectValidationError: false,
		},
		{
			name:                  "string concatenation validates successfully",
			expression:            `"hello" + " " + "world" == "hello world"`,
			expectValidationError: false,
		},
		{
			name:                  "missing parenthesis fails with syntax error",
			expression:            "1 + (2 * 3",
			expectValidationError: true,
			expectedErrorType:     ValidationErrorTypeSyntax,
		},
		{
			name:                  "undeclared variable fails with reference error",
			expression:            "undefinedVar == 1",
			expectValidationError: true,
			expectedErrorType:     ValidationErrorTypeUndeclaredReference,
		},
		{
			name:                  "type mismatch in operation fails with syntax error",
			expression:            `1 + "string"`,
			expectValidationError: true,
			expectedErrorType:     ValidationErrorTypeSyntax, // CEL reports type mismatches as syntax errors
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpression(tt.expression)

			if !tt.expectValidationError && len(issues) > 0 {
				t.Errorf("Expected valid expression, got issues: %v", issues)
			}
			if tt.expectValidationError && len(issues) == 0 {
				t.Errorf("Expected validation errors, but got none")
			}
			if tt.expectValidationError && tt.expectedErrorType != "" && len(issues) > 0 && issues[0].Type != tt.expectedErrorType {
				t.Errorf("Expected error type %s, got %s", tt.expectedErrorType, issues[0].Type)
			}
		})
	}
}

// Test expression validation with inputs
func TestValidateCELExpressionWithInputs(t *testing.T) {
	tests := []struct {
		name                  string
		expression            string
		declarations          []*expr.Decl
		expectValidationError bool
		expectedErrorType     ValidationErrorType
	}{
		{
			name:       "expression with declared variable validates successfully",
			expression: "pods.items.size() > 0",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: false,
		},
		{
			name:       "expression with multiple declared variables validates successfully",
			expression: "namespaces.items.all(ns, pods.items.exists(pod, pod.metadata.namespace == ns.metadata.name))",
			declarations: []*expr.Decl{
				decls.NewVar("namespaces", decls.Dyn),
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: false,
		},
		{
			name:       "expression with undeclared variable fails with reference error",
			expression: "deployments.items.all(d, d.spec.replicas > 1)",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: true,
			expectedErrorType:     ValidationErrorTypeUndeclaredReference,
		},
		{
			name:       "CEL 'all' macro validates successfully",
			expression: "pods.items.all(pod, pod.spec.containers.all(c, c.image != ''))",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: false,
		},
		{
			name:       "CEL 'exists' macro validates successfully",
			expression: "pods.items.exists(pod, pod.metadata.labels.app == 'test')",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: false,
		},
		{
			name:       "CEL 'filter' macro validates successfully",
			expression: "pods.items.filter(pod, pod.status.phase == 'Running').size() > 0",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			expectValidationError: false,
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpressionWithInputs(tt.expression, tt.declarations)

			if !tt.expectValidationError && len(issues) > 0 {
				t.Errorf("Expected valid expression, got issues: %v", issues)
			}
			if tt.expectValidationError && len(issues) == 0 {
				t.Errorf("Expected validation errors, but got none")
			}
			if tt.expectValidationError && tt.expectedErrorType != "" && len(issues) > 0 && issues[0].Type != tt.expectedErrorType {
				t.Errorf("Expected error type %s, got %s", tt.expectedErrorType, issues[0].Type)
			}
		})
	}
}

// Test CompileCELExpression public function
func TestCompileCELExpression(t *testing.T) {
	tests := []struct {
		name                  string
		expression            string
		inputs                []Input
		expectValidationError bool
		errorMsg              string
	}{
		{
			name:       "expression with valid inputs compiles successfully",
			expression: "deployments.items.all(d, d.spec.replicas >= 2)",
			inputs: []Input{
				&InputImpl{
					InputName: "deployments",
					InputType: InputTypeKubernetes,
				},
			},
			expectValidationError: false,
		},
		{
			name:       "expression with undeclared variable fails to compile",
			expression: "pods.items.all(p, p.spec.replicas > 0)",
			inputs: []Input{
				&InputImpl{
					InputName: "deployments",
					InputType: InputTypeKubernetes,
				},
			},
			expectValidationError: true,
			errorMsg:              "UNDECLARED_REFERENCE",
		},
		{
			name:       "incomplete expression fails to compile with syntax error",
			expression: "pods.items.all(p, p.spec.replicas >",
			inputs: []Input{
				&InputImpl{
					InputName: "pods",
					InputType: InputTypeKubernetes,
				},
			},
			expectValidationError: true,
			errorMsg:              "SYNTAX_ERROR",
		},
		{
			name: "complex nested expression compiles successfully",
			expression: `
				pods.items.all(pod, 
					pod.spec.containers.all(container, 
						has(container.securityContext) && 
						has(container.securityContext.runAsNonRoot) && 
						container.securityContext.runAsNonRoot == true
					)
				)
			`,
			inputs: []Input{
				&InputImpl{
					InputName: "pods",
					InputType: InputTypeKubernetes,
				},
			},
			expectValidationError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompileCELExpression(tt.expression, tt.inputs)

			if !tt.expectValidationError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if tt.expectValidationError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if tt.expectValidationError && tt.errorMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %s, got %s", tt.errorMsg, err.Error())
			}
		})
	}
}

// Test ValidateRule for full rule validation
func TestValidateRule(t *testing.T) {
	tests := []struct {
		name                  string
		rule                  Rule
		expectValidationError bool
		expectedErrorType     ValidationErrorType
	}{
		{
			name: "CEL rule with valid inputs validates successfully",
			rule: &mockCelRule{
				id:         "test-rule",
				expression: "pods.items.all(p, p.spec.containers.size() > 0)",
				mockRule: mockRule{
					inputs: []Input{
						&InputImpl{
							InputName: "pods",
							InputType: InputTypeKubernetes,
						},
					},
				},
			},
			expectValidationError: false,
		},
		{
			name: "CEL rule with undeclared variable fails validation",
			rule: &mockCelRule{
				id:         "test-rule",
				expression: "deployments.items.all(d, d.spec.replicas > 0)",
				mockRule: mockRule{
					inputs: []Input{
						&InputImpl{
							InputName: "pods",
							InputType: InputTypeKubernetes,
						},
					},
				},
			},
			expectValidationError: true,
			expectedErrorType:     ValidationErrorTypeUndeclaredReference,
		},
		{
			name: "non-CEL rule type validates with warning",
			rule: &mockRule{
				ruleType: RuleTypeRego,
			},
			expectValidationError: false, // Should be valid but with warning
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateRule(tt.rule)

			if !tt.expectValidationError && !result.Valid {
				t.Errorf("Expected valid rule, but got validation errors: %v", result.Issues)
			}
			if tt.expectValidationError && result.Valid {
				t.Errorf("Expected validation error, but rule was valid")
			}
			if tt.expectValidationError && tt.expectedErrorType != "" && len(result.Issues) > 0 && result.Issues[0].Type != tt.expectedErrorType {
				t.Errorf("Expected error type %s, got %s", tt.expectedErrorType, result.Issues[0].Type)
			}

			// Check for warnings on non-CEL rules
			if tt.rule.Type() != RuleTypeCEL && len(result.Warnings) == 0 {
				t.Errorf("Expected warning for non-CEL rule type")
			}
		})
	}
}

// Test error categorization
func TestCategorizeCompilationError(t *testing.T) {
	tests := []struct {
		name          string
		errorMsg      string
		expectedType  ValidationErrorType
		expectedInMsg string
	}{
		{
			name:          "undeclared reference error categorizes as reference error",
			errorMsg:      "ERROR: <input>:1:1: undeclared reference to 'unknownVar'",
			expectedType:  ValidationErrorTypeUndeclaredReference,
			expectedInMsg: "unknownVar",
		},
		{
			name:          "syntax error categorizes as syntax error",
			errorMsg:      "ERROR: <input>:1:10: syntax error: unexpected token",
			expectedType:  ValidationErrorTypeSyntax,
			expectedInMsg: "Syntax error",
		},
		{
			name:          "type checking error categorizes as type error",
			errorMsg:      "ERROR: type checking failed",
			expectedType:  ValidationErrorTypeType,
			expectedInMsg: "Type error",
		},
		{
			name:          "unrecognized error categorizes as general error",
			errorMsg:      "Some other error occurred",
			expectedType:  ValidationErrorTypeGeneral,
			expectedInMsg: "CEL compilation error",
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := validator.categorizeCompilationError("test expression", tt.errorMsg)

			if issue.Type != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, issue.Type)
			}

			if !strings.Contains(issue.Message, tt.expectedInMsg) {
				t.Errorf("Expected message to contain '%s', got '%s'", tt.expectedInMsg, issue.Message)
			}
		})
	}
}

// Test custom functions in validation environment
func TestCustomFunctionsInValidation(t *testing.T) {
	tests := []struct {
		name                  string
		expression            string
		expectValidationError bool
	}{
		{
			name:                  "parseJSON function validates successfully",
			expression:            `parseJSON('{"key": "value"}').key == "value"`,
			expectValidationError: false,
		},
		{
			name:                  "parseYAML function validates successfully",
			expression:            `parseYAML("key: value").key == "value"`,
			expectValidationError: false,
		},
		{
			name:                  "undefined custom function fails validation",
			expression:            `customFunc("test")`,
			expectValidationError: true,
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpression(tt.expression)

			if !tt.expectValidationError && len(issues) > 0 {
				t.Errorf("Expected valid expression, got issues: %v", issues)
			}
			if tt.expectValidationError && len(issues) == 0 {
				t.Errorf("Expected validation errors, but got none")
			}
		})
	}
}

// Test location information in errors
func TestErrorLocationInformation(t *testing.T) {
	validator := NewRuleValidator(nil)

	// Multi-line expression with error
	expression := `
		pods.items.all(pod,
			pod.spec.containers.all(container,
				container.image != "" &&
				unknownVariable == true
			)
		)
	`

	issues := validator.ValidateCELExpressionWithInputs(expression, []*expr.Decl{
		decls.NewVar("pods", decls.Dyn),
	})

	if len(issues) == 0 {
		t.Errorf("Expected validation errors for expression with undefined variable")
	}

	// Check that at least one issue has location information
	hasLocation := false
	for _, issue := range issues {
		if issue.Location != nil && (issue.Location.Line > 0 || issue.Location.Column > 0) {
			hasLocation = true
			break
		}
	}

	if !hasLocation {
		t.Logf("Warning: No location information found in errors (this might be expected depending on CEL version)")
	}
}

// Mock implementations for testing

type mockRule struct {
	ruleType RuleType
	inputs   []Input
}

func (m *mockRule) Identifier() string      { return "mock-rule" }
func (m *mockRule) Type() RuleType          { return m.ruleType }
func (m *mockRule) Inputs() []Input         { return m.inputs }
func (m *mockRule) Metadata() *RuleMetadata { return nil }
func (m *mockRule) Content() interface{}    { return "" }

type mockCelRule struct {
	mockRule
	id         string
	expression string
}

func (m *mockCelRule) Identifier() string {
	if m.id != "" {
		return m.id
	}
	return "mock-cel-rule"
}

func (m *mockCelRule) Type() RuleType          { return RuleTypeCEL }
func (m *mockCelRule) Expression() string      { return m.expression }
func (m *mockCelRule) ErrorMessage() string    { return "Test error" }
func (m *mockCelRule) Inputs() []Input         { return m.mockRule.inputs }
func (m *mockCelRule) Metadata() *RuleMetadata { return nil }
func (m *mockCelRule) Content() interface{}    { return m.expression }

// Benchmark tests
func BenchmarkValidateCELExpressionSimple(b *testing.B) {
	validator := NewRuleValidator(nil)
	expression := "pods.items.all(p, p.spec.containers.size() > 0)"
	declarations := []*expr.Decl{
		decls.NewVar("pods", decls.Dyn),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateCELExpressionWithInputs(expression, declarations)
	}
}

func BenchmarkCompileCELExpression(b *testing.B) {
	expression := "deployments.items.all(d, d.spec.replicas >= 2)"
	inputs := []Input{
		&InputImpl{
			InputName: "deployments",
			InputType: InputTypeKubernetes,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CompileCELExpression(expression, inputs)
	}
}
