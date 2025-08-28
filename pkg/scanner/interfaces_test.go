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
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestCelRuleInterface validates the CelRule interface implementation
func TestCelRuleInterface(t *testing.T) {
	tests := []struct {
		name               string
		rule               CelRule
		expectedID         string
		expectedExpr       string
		expectedInputCount int
		hasMetadata        bool
	}{
		{
			name: "simple rule without metadata",
			rule: NewCelRule("test-001", "pods.items.size() > 0", []Input{
				NewKubernetesInput("pods", "", "v1", "pods", "", ""),
			}),
			expectedID:         "test-001",
			expectedExpr:       "pods.items.size() > 0",
			expectedInputCount: 1,
			hasMetadata:        false,
		},
		{
			name: "rule with metadata",
			rule: NewCelRuleWithMetadata(
				"test-002",
				"pods.items.all(pod, has(pod.spec.securityContext))",
				[]Input{
					NewKubernetesInput("pods", "", "v1", "pods", "", ""),
				},
				&RuleMetadata{
					Name:        "Security Context Check",
					Description: "Ensures all pods have security context",
				},
			),
			expectedID:         "test-002",
			expectedExpr:       "pods.items.all(pod, has(pod.spec.securityContext))",
			expectedInputCount: 1,
			hasMetadata:        true,
		},
		{
			name: "rule with multiple inputs",
			rule: NewCelRule("test-003", "pods.items.size() > 0 && config.enabled", []Input{
				NewKubernetesInput("pods", "", "v1", "pods", "", ""),
				NewFileInput("config", "/etc/config.yaml", "yaml", false, false),
			}),
			expectedID:         "test-003",
			expectedExpr:       "pods.items.size() > 0 && config.enabled",
			expectedInputCount: 2,
			hasMetadata:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test CelRule interface methods
			if tt.rule.Identifier() != tt.expectedID {
				t.Errorf("Expected ID %s, got %s", tt.expectedID, tt.rule.Identifier())
			}

			if tt.rule.Expression() != tt.expectedExpr {
				t.Errorf("Expected expression %s, got %s", tt.expectedExpr, tt.rule.Expression())
			}

			inputs := tt.rule.Inputs()
			if len(inputs) != tt.expectedInputCount {
				t.Errorf("Expected %d inputs, got %d", tt.expectedInputCount, len(inputs))
			}

			metadata := tt.rule.Metadata()
			if tt.hasMetadata && metadata == nil {
				t.Error("Expected metadata but got nil")
			}
			if !tt.hasMetadata && metadata != nil {
				t.Error("Expected no metadata but got metadata")
			}

			if tt.hasMetadata && metadata != nil {
				if metadata.Name == "" {
					t.Error("Expected metadata name to be set")
				}
			}
		})
	}
}

// TestRuleBuilder validates the fluent builder pattern
func TestRuleBuilder(t *testing.T) {
	t.Run("fluent builder", func(t *testing.T) {
		rule, err := NewRuleBuilder("builder-test", RuleTypeCEL).
			WithKubernetesInput("pods", "", "v1", "pods", "", "").
			WithFileInput("config", "/etc/config.yaml", "yaml", false, false).
			WithSystemInput("service", "nginx", "", []string{}).
			SetCelExpression("pods.items.size() > 0").
			WithName("Test Rule").
			WithDescription("Test rule for builder").
			BuildCelRule()
		if err != nil {
			t.Fatalf("Failed to build rule: %v", err)
		}

		if rule.Identifier() != "builder-test" {
			t.Errorf("Expected ID 'builder-test', got %s", rule.Identifier())
		}

		if rule.Expression() != "pods.items.size() > 0" {
			t.Errorf("Expected expression 'pods.items.size() > 0', got %s", rule.Expression())
		}

		inputs := rule.Inputs()
		if len(inputs) != 3 {
			t.Errorf("Expected 3 inputs, got %d", len(inputs))
		}

		expectedNames := []string{"pods", "config", "service"}
		for i, input := range inputs {
			if input.Name() != expectedNames[i] {
				t.Errorf("Expected input %d to be %s, got %s", i, expectedNames[i], input.Name())
			}
		}

		metadata := rule.Metadata()
		if metadata == nil {
			t.Fatal("Expected metadata to be set")
		}

		if metadata.Name != "Test Rule" {
			t.Errorf("Expected name 'Test Rule', got %s", metadata.Name)
		}

		if metadata.Description != "Test rule for builder" {
			t.Errorf("Expected description 'Test rule for builder', got %s", metadata.Description)
		}
	})
}

// TestInputTypes validates all input type implementations
func TestInputTypes(t *testing.T) {
	tests := []struct {
		name         string
		input        Input
		expectedName string
		expectedType InputType
		specTest     func(t *testing.T, spec InputSpec)
	}{
		{
			name:         "kubernetes input",
			input:        NewKubernetesInput("pods", "apps", "v1", "deployments", "default", "nginx"),
			expectedName: "pods",
			expectedType: InputTypeKubernetes,
			specTest: func(t *testing.T, spec InputSpec) {
				kubeSpec, ok := spec.(KubernetesInputSpec)
				if !ok {
					t.Fatal("Expected KubernetesInputSpec")
				}
				if kubeSpec.ApiGroup() != "apps" {
					t.Errorf("Expected API group 'apps', got %s", kubeSpec.ApiGroup())
				}
				if kubeSpec.Version() != "v1" {
					t.Errorf("Expected version 'v1', got %s", kubeSpec.Version())
				}
				if kubeSpec.ResourceType() != "deployments" {
					t.Errorf("Expected resource type 'deployments', got %s", kubeSpec.ResourceType())
				}
				if kubeSpec.Namespace() != "default" {
					t.Errorf("Expected namespace 'default', got %s", kubeSpec.Namespace())
				}
				if kubeSpec.Name() != "nginx" {
					t.Errorf("Expected name 'nginx', got %s", kubeSpec.Name())
				}
				if err := kubeSpec.Validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			},
		},
		{
			name:         "file input",
			input:        NewFileInput("config", "/etc/config.yaml", "yaml", true, true),
			expectedName: "config",
			expectedType: InputTypeFile,
			specTest: func(t *testing.T, spec InputSpec) {
				fileSpec, ok := spec.(FileInputSpec)
				if !ok {
					t.Fatal("Expected FileInputSpec")
				}
				if fileSpec.Path() != "/etc/config.yaml" {
					t.Errorf("Expected path '/etc/config.yaml', got %s", fileSpec.Path())
				}
				if fileSpec.Format() != "yaml" {
					t.Errorf("Expected format 'yaml', got %s", fileSpec.Format())
				}
				if !fileSpec.Recursive() {
					t.Error("Expected recursive to be true")
				}
				if !fileSpec.CheckPermissions() {
					t.Error("Expected check permissions to be true")
				}
				if err := fileSpec.Validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			},
		},
		{
			name:         "system input",
			input:        NewSystemInput("nginx", "nginx", "", []string{}),
			expectedName: "nginx",
			expectedType: InputTypeSystem,
			specTest: func(t *testing.T, spec InputSpec) {
				systemSpec, ok := spec.(SystemInputSpec)
				if !ok {
					t.Fatal("Expected SystemInputSpec")
				}
				if systemSpec.ServiceName() != "nginx" {
					t.Errorf("Expected service name 'nginx', got %s", systemSpec.ServiceName())
				}
				if systemSpec.Command() != "" {
					t.Errorf("Expected empty command, got %s", systemSpec.Command())
				}
				if len(systemSpec.Args()) != 0 {
					t.Errorf("Expected no args, got %v", systemSpec.Args())
				}
				if err := systemSpec.Validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			},
		},
		{
			name:         "http input",
			input:        NewHTTPInput("api", "https://api.example.com", "GET", map[string]string{"Authorization": "Bearer token"}, nil),
			expectedName: "api",
			expectedType: InputTypeHTTP,
			specTest: func(t *testing.T, spec InputSpec) {
				httpSpec, ok := spec.(HTTPInputSpec)
				if !ok {
					t.Fatal("Expected HTTPInputSpec")
				}
				if httpSpec.URL() != "https://api.example.com" {
					t.Errorf("Expected URL 'https://api.example.com', got %s", httpSpec.URL())
				}
				if httpSpec.Method() != "GET" {
					t.Errorf("Expected method 'GET', got %s", httpSpec.Method())
				}
				headers := httpSpec.Headers()
				if len(headers) != 1 {
					t.Errorf("Expected 1 header, got %d", len(headers))
				}
				if headers["Authorization"] != "Bearer token" {
					t.Errorf("Expected Authorization header, got %v", headers)
				}
				if httpSpec.Body() != nil {
					t.Errorf("Expected nil body, got %v", httpSpec.Body())
				}
				if err := httpSpec.Validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.Name() != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, tt.input.Name())
			}

			if tt.input.Type() != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, tt.input.Type())
			}

			spec := tt.input.Spec()
			if spec == nil {
				t.Fatal("Expected spec but got nil")
			}

			tt.specTest(t, spec)
		})
	}
}

// TestRuleMetadata validates simplified rule metadata functionality
func TestRuleMetadata(t *testing.T) {
	metadata := &RuleMetadata{
		Name:        "Test Rule",
		Description: "A test rule for validation",
		Extensions:  map[string]interface{}{"custom": "value"},
	}

	rule := NewCelRuleWithMetadata("meta-test", "true", []Input{}, metadata)

	returnedMetadata := rule.Metadata()
	if returnedMetadata == nil {
		t.Fatal("Expected metadata but got nil")
	}

	// Test simplified metadata fields
	if returnedMetadata.Name != "Test Rule" {
		t.Errorf("Expected name 'Test Rule', got %s", returnedMetadata.Name)
	}

	if returnedMetadata.Description != "A test rule for validation" {
		t.Errorf("Expected description 'A test rule for validation', got %s", returnedMetadata.Description)
	}

	if returnedMetadata.Extensions["custom"] != "value" {
		t.Errorf("Expected extension custom=value, got %v", returnedMetadata.Extensions)
	}
}

// TestBuilderChaining validates that builder methods return the builder for chaining
func TestBuilderChaining(t *testing.T) {
	builder := NewRuleBuilder("chain-test", RuleTypeCEL)

	// Test that all methods return the builder
	result := builder.
		WithInput(NewKubernetesInput("pods", "", "v1", "pods", "", "")).
		WithKubernetesInput("services", "", "v1", "services", "", "").
		WithFileInput("config", "/etc/config.yaml", "yaml", false, false).
		WithSystemInput("nginx", "nginx", "", []string{}).
		SetCelExpression("true").
		WithName("Chained Rule").
		WithDescription("Tests method chaining")

	// Verify it's still the same builder
	if result != builder {
		t.Error("Builder methods should return the same builder instance")
	}

	rule, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build rule: %v", err)
	}
	if len(rule.Inputs()) != 4 {
		t.Errorf("Expected 4 inputs after chaining, got %d", len(rule.Inputs()))
	}

	metadata := rule.Metadata()
	if metadata.Name != "Chained Rule" {
		t.Errorf("Expected name 'Chained Rule', got %s", metadata.Name)
	}
}

// Mock implementation for testing CelVariable interface
type MockCelVariable struct {
	name      string
	namespace string
	value     string
	gvk       schema.GroupVersionKind
}

func (m *MockCelVariable) Name() string                              { return m.name }
func (m *MockCelVariable) Namespace() string                         { return m.namespace }
func (m *MockCelVariable) Value() string                             { return m.value }
func (m *MockCelVariable) GroupVersionKind() schema.GroupVersionKind { return m.gvk }

// TestCelVariableInterface validates the CelVariable interface
func TestCelVariableInterface(t *testing.T) {
	// Verify interface compliance
	var _ CelVariable = &MockCelVariable{}

	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	variable := &MockCelVariable{
		name:      "replicas",
		namespace: "default",
		value:     "3",
		gvk:       gvk,
	}

	if variable.Name() != "replicas" {
		t.Errorf("Expected name 'replicas', got %s", variable.Name())
	}

	if variable.Namespace() != "default" {
		t.Errorf("Expected namespace 'default', got %s", variable.Namespace())
	}

	if variable.Value() != "3" {
		t.Errorf("Expected value '3', got %s", variable.Value())
	}

	returnedGVK := variable.GroupVersionKind()
	if returnedGVK.Group != "apps" || returnedGVK.Version != "v1" || returnedGVK.Kind != "Deployment" {
		t.Errorf("Expected GVK apps/v1/Deployment, got %v", returnedGVK)
	}
}

// TestInterfaceCompliance validates that all implementations satisfy their interfaces
func TestInterfaceCompliance(t *testing.T) {
	// Verify all interface implementations compile
	var _ CelRule = &CelRuleImpl{}
	var _ Input = &InputImpl{}
	var _ InputSpec = &KubernetesInput{}
	var _ InputSpec = &FileInput{}
	var _ InputSpec = &SystemInput{}
	var _ InputSpec = &HTTPInput{}
	var _ KubernetesInputSpec = &KubernetesInput{}
	var _ FileInputSpec = &FileInput{}
	var _ SystemInputSpec = &SystemInput{}
	var _ HTTPInputSpec = &HTTPInput{}
	var _ CelVariable = &MockCelVariable{}
}
