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

package main

import (
	"fmt"

	"github.com/ComplianceAsCode/compliance-sdk/pkg/scanner"
)

func main() {
	// Example 1: Simple validation of a CEL expression
	fmt.Println("=== Example 1: Simple CEL Expression Validation ===")
	validateSimpleExpression()

	// Example 2: Validation with input declarations
	fmt.Println("\n=== Example 2: CEL Expression with Inputs ===")
	validateWithInputs()

	// Example 3: Validation using RuleValidator
	fmt.Println("\n=== Example 3: Full Rule Validation ===")
	validateFullRule()

	// Example 4: Validate an invalid expression
	fmt.Println("\n=== Example 4: Invalid Expression ===")
	validateInvalidExpression()
}

func validateSimpleExpression() {
	// Validate a simple expression without inputs
	expression := `1 + 1 == 2`

	validator := scanner.NewRuleValidator(nil)
	issues := validator.ValidateCELExpressionSimple(expression)

	if len(issues) == 0 {
		fmt.Println("✅ Expression is valid:", expression)
	} else {
		fmt.Println("❌ Expression has validation errors:")
		for _, issue := range issues {
			fmt.Printf("  - %s: %s\n", issue.Type, issue.Message)
		}
	}
}

func validateWithInputs() {
	// Create inputs for a Kubernetes resource check
	inputs := []scanner.Input{
		&scanner.InputImpl{
			InputName: "pods",
			InputType: scanner.InputTypeKubernetes,
			InputSpec: &KubernetesSpec{
				APIVersion: "v1",
				Resource:   "pods",
			},
		},
		&scanner.InputImpl{
			InputName: "namespaces",
			InputType: scanner.InputTypeKubernetes,
			InputSpec: &KubernetesSpec{
				APIVersion: "v1",
				Resource:   "namespaces",
			},
		},
	}

	// Expression that checks all pods have non-root containers
	expression := `
		pods.items.all(pod, 
			pod.spec.containers.all(container, 
				container.securityContext.runAsNonRoot == true
			)
		)
	`

	// Validate the expression with inputs
	err := scanner.CompileCELExpression(expression, inputs)
	if err != nil {
		fmt.Printf("❌ Expression validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Expression is valid with the provided inputs")
	}
}

func validateFullRule() {
	// Create a rule using the builder
	builder := scanner.NewRuleBuilder("test-rule", scanner.RuleTypeCEL)

	// Add inputs
	builder.WithInput(&scanner.InputImpl{
		InputName: "deployments",
		InputType: scanner.InputTypeKubernetes,
		InputSpec: &KubernetesSpec{
			APIVersion: "apps/v1",
			Resource:   "deployments",
			Group:      "apps",
			Namespace:  "production",
		},
	})

	// Set the expression
	builder.SetCelExpression(`
		deployments.items.all(deploy,
			deploy.spec.replicas >= 2
		)
	`)

	// Build the rule
	rule, err := builder.BuildCelRule()
	if err != nil {
		fmt.Printf("❌ Failed to build rule: %v\n", err)
		return
	}

	// Validate the rule
	validator := scanner.NewRuleValidator(nil)
	result := validator.ValidateRule(rule)

	if result.Valid {
		fmt.Println("✅ Rule is valid")
	} else {
		fmt.Println("❌ Rule validation failed:")
		for _, issue := range result.Issues {
			fmt.Printf("  - [%s] %s", issue.Type, issue.Message)
			if issue.Details != "" {
				fmt.Printf(" (%s)", issue.Details)
			}
			if issue.Location != nil {
				fmt.Printf(" at line %d, col %d", issue.Location.Line, issue.Location.Column)
			}
			fmt.Println()
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("⚠️ Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
}

func validateInvalidExpression() {
	// Create inputs
	inputs := []scanner.Input{
		&scanner.InputImpl{
			InputName: "pods",
			InputType: scanner.InputTypeKubernetes,
			InputSpec: &KubernetesSpec{
				APIVersion: "v1",
				Resource:   "pods",
			},
		},
	}

	// Expression with an undeclared variable
	expression := `
		pods.items.all(pod, 
			pod.spec.containers.all(container, 
				container.name == undeclaredVariable
			)
		)
	`

	// Validate the expression
	err := scanner.CompileCELExpression(expression, inputs)
	if err != nil {
		fmt.Printf("❌ Expected validation error: %v\n", err)
	} else {
		fmt.Println("⚠️ Expression was unexpectedly valid")
	}

	// Get detailed validation information
	validator := scanner.NewRuleValidator(nil)
	issues := validator.ValidateCELExpression(expression, nil)

	fmt.Println("\nDetailed validation issues:")
	for _, issue := range issues {
		fmt.Printf("  Type: %s\n", issue.Type)
		fmt.Printf("  Message: %s\n", issue.Message)
		if issue.Details != "" {
			fmt.Printf("  Details: %s\n", issue.Details)
		}
		if issue.Location != nil {
			fmt.Printf("  Location: Line %d, Column %d\n",
				issue.Location.Line, issue.Location.Column)
		}
		fmt.Println()
	}
}

// KubernetesSpec is a simple implementation for the example
type KubernetesSpec struct {
	APIVersion string
	Resource   string
	Group      string
	Namespace  string
}

func (k *KubernetesSpec) GetApiVersion() string { return k.APIVersion }
func (k *KubernetesSpec) GetResource() string   { return k.Resource }
func (k *KubernetesSpec) GetGroup() string      { return k.Group }
func (k *KubernetesSpec) GetNamespace() string  { return k.Namespace }
func (k *KubernetesSpec) GetName() string       { return "" }
func (k *KubernetesSpec) Validate() error {
	if k.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if k.Resource == "" {
		return fmt.Errorf("resource is required")
	}
	return nil
}
func (k *KubernetesSpec) GetGVR() (string, string, string) {
	return k.Group, k.APIVersion, k.Resource
}
