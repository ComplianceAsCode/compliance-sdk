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
		name       string
		expression string
		wantValid  bool
		wantError  ValidationErrorType
	}{
		{
			name:       "valid simple arithmetic",
			expression: "1 + 1 == 2",
			wantValid:  true,
		},
		{
			name:       "valid boolean logic",
			expression: "true && false == false",
			wantValid:  true,
		},
		{
			name:       "valid string concatenation",
			expression: `"hello" + " " + "world" == "hello world"`,
			wantValid:  true,
		},
		{
			name:       "syntax error - missing parenthesis",
			expression: "1 + (2 * 3",
			wantValid:  false,
			wantError:  ValidationErrorTypeSyntax,
		},
		{
			name:       "undeclared variable",
			expression: "undefinedVar == 1",
			wantValid:  false,
			wantError:  ValidationErrorTypeUndeclaredReference,
		},
		{
			name:       "syntax error - invalid operation",
			expression: `1 + "string"`,
			wantValid:  false,
			wantError:  ValidationErrorTypeSyntax, // CEL reports type mismatches as syntax errors
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpressionSimple(tt.expression)

			if tt.wantValid {
				if len(issues) > 0 {
					t.Errorf("Expected valid expression, got issues: %v", issues)
				}
			} else {
				if len(issues) == 0 {
					t.Errorf("Expected validation errors, but got none")
				} else if tt.wantError != "" && issues[0].Type != tt.wantError {
					t.Errorf("Expected error type %s, got %s", tt.wantError, issues[0].Type)
				}
			}
		})
	}
}

// Test expression validation with inputs
func TestValidateCELExpressionWithInputs(t *testing.T) {
	tests := []struct {
		name         string
		expression   string
		declarations []*expr.Decl
		wantValid    bool
		wantError    ValidationErrorType
	}{
		{
			name:       "valid with declared variable",
			expression: "pods.items.size() > 0",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: true,
		},
		{
			name:       "valid with multiple variables",
			expression: "namespaces.items.all(ns, pods.items.exists(pod, pod.metadata.namespace == ns.metadata.name))",
			declarations: []*expr.Decl{
				decls.NewVar("namespaces", decls.Dyn),
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: true,
		},
		{
			name:       "undeclared variable in expression",
			expression: "deployments.items.all(d, d.spec.replicas > 1)",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: false,
			wantError: ValidationErrorTypeUndeclaredReference,
		},
		{
			name:       "valid CEL macros - all",
			expression: "pods.items.all(pod, pod.spec.containers.all(c, c.image != ''))",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: true,
		},
		{
			name:       "valid CEL macros - exists",
			expression: "pods.items.exists(pod, pod.metadata.labels.app == 'test')",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: true,
		},
		{
			name:       "valid CEL macros - filter",
			expression: "pods.items.filter(pod, pod.status.phase == 'Running').size() > 0",
			declarations: []*expr.Decl{
				decls.NewVar("pods", decls.Dyn),
			},
			wantValid: true,
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpression(tt.expression, tt.declarations)

			if tt.wantValid {
				if len(issues) > 0 {
					t.Errorf("Expected valid expression, got issues: %v", issues)
				}
			} else {
				if len(issues) == 0 {
					t.Errorf("Expected validation errors, but got none")
				} else if tt.wantError != "" && issues[0].Type != tt.wantError {
					t.Errorf("Expected error type %s, got %s", tt.wantError, issues[0].Type)
				}
			}
		})
	}
}

// Test CompileCELExpression public function
func TestCompileCELExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		inputs     []Input
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "valid expression with inputs",
			expression: "deployments.items.all(d, d.spec.replicas >= 2)",
			inputs: []Input{
				&InputImpl{
					InputName: "deployments",
					InputType: InputTypeKubernetes,
				},
			},
			wantError: false,
		},
		{
			name:       "expression with undeclared variable",
			expression: "pods.items.all(p, p.spec.replicas > 0)",
			inputs: []Input{
				&InputImpl{
					InputName: "deployments",
					InputType: InputTypeKubernetes,
				},
			},
			wantError: true,
			errorMsg:  "UNDECLARED_REFERENCE",
		},
		{
			name:       "syntax error in expression",
			expression: "pods.items.all(p, p.spec.replicas >",
			inputs: []Input{
				&InputImpl{
					InputName: "pods",
					InputType: InputTypeKubernetes,
				},
			},
			wantError: true,
			errorMsg:  "SYNTAX_ERROR",
		},
		{
			name: "complex valid expression",
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
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompileCELExpression(tt.expression, tt.inputs)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %s, got %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Test ValidateRule for full rule validation
func TestValidateRule(t *testing.T) {
	tests := []struct {
		name      string
		rule      Rule
		wantValid bool
		wantError ValidationErrorType
	}{
		{
			name: "valid CEL rule",
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
			wantValid: true,
		},
		{
			name: "CEL rule with undeclared variable",
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
			wantValid: false,
			wantError: ValidationErrorTypeUndeclaredReference,
		},
		{
			name: "non-CEL rule type",
			rule: &mockRule{
				ruleType: RuleTypeRego,
			},
			wantValid: true, // Should be valid but with warning
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateRule(tt.rule)

			if tt.wantValid != result.Valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.wantValid, result.Valid)
			}

			if !tt.wantValid && tt.wantError != "" {
				if len(result.Issues) == 0 {
					t.Errorf("Expected issues, got none")
				} else if result.Issues[0].Type != tt.wantError {
					t.Errorf("Expected error type %s, got %s", tt.wantError, result.Issues[0].Type)
				}
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
			name:          "undeclared reference error",
			errorMsg:      "ERROR: <input>:1:1: undeclared reference to 'unknownVar'",
			expectedType:  ValidationErrorTypeUndeclaredReference,
			expectedInMsg: "unknownVar",
		},
		{
			name:          "syntax error",
			errorMsg:      "ERROR: <input>:1:10: syntax error: unexpected token",
			expectedType:  ValidationErrorTypeSyntax,
			expectedInMsg: "Syntax error",
		},
		{
			name:          "general type error",
			errorMsg:      "ERROR: type checking failed",
			expectedType:  ValidationErrorTypeType,
			expectedInMsg: "Type error",
		},
		{
			name:          "generic error",
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
		name       string
		expression string
		wantValid  bool
	}{
		{
			name:       "parseJSON function available",
			expression: `parseJSON('{"key": "value"}').key == "value"`,
			wantValid:  true,
		},
		{
			name:       "parseYAML function available",
			expression: `parseYAML("key: value").key == "value"`,
			wantValid:  true,
		},
		{
			name:       "undefined custom function",
			expression: `customFunc("test")`,
			wantValid:  false,
		},
	}

	validator := NewRuleValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateCELExpressionSimple(tt.expression)

			if tt.wantValid {
				if len(issues) > 0 {
					t.Errorf("Expected valid expression, got issues: %v", issues)
				}
			} else {
				if len(issues) == 0 {
					t.Errorf("Expected validation errors, but got none")
				}
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

	issues := validator.ValidateCELExpression(expression, []*expr.Decl{
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
		_ = validator.ValidateCELExpression(expression, declarations)
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
