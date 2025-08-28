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

package fetchers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ComplianceAsCode/compliance-sdk/pkg/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Mock implementations for testing

type mockInputFetcher struct {
	supportedTypes []scanner.InputType
	fetchData      map[string]interface{}
	fetchError     error
}

func (m *mockInputFetcher) FetchInputs(inputs []scanner.Input, variables []scanner.CelVariable) (map[string]interface{}, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	return m.fetchData, nil
}

func (m *mockInputFetcher) SupportsInputType(inputType scanner.InputType) bool {
	for _, supported := range m.supportedTypes {
		if supported == inputType {
			return true
		}
	}
	return false
}

type mockRule struct {
	identifier string
	inputs     []scanner.Input
	ruleType   scanner.RuleType
}

func (m *mockRule) Identifier() string              { return m.identifier }
func (m *mockRule) Type() scanner.RuleType          { return m.ruleType }
func (m *mockRule) Inputs() []scanner.Input         { return m.inputs }
func (m *mockRule) Metadata() *scanner.RuleMetadata { return &scanner.RuleMetadata{} }
func (m *mockRule) Content() interface{}            { return "test-content" }

type mockInput struct {
	name      string
	inputType scanner.InputType
	spec      scanner.InputSpec
}

func (m *mockInput) Name() string            { return m.name }
func (m *mockInput) Type() scanner.InputType { return m.inputType }
func (m *mockInput) Spec() scanner.InputSpec { return m.spec }

type mockInputSpec struct {
	valid bool
}

func (m *mockInputSpec) Validate() error {
	if !m.valid {
		return errors.New("invalid spec")
	}
	return nil
}

func TestNewCompositeFetcher(t *testing.T) {
	t.Run("creates empty composite fetcher", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		assert.NotNil(t, fetcher)
		assert.NotNil(t, fetcher.customFetchers)
		assert.Equal(t, 0, len(fetcher.customFetchers))
	})
}

func TestNewCompositeFetcherWithDefaults(t *testing.T) {
	t.Run("creates fetcher with defaults", func(t *testing.T) {
		fetcher := NewCompositeFetcherWithDefaults(
			nil,
			nil,
			"/tmp/api-resources",
			"/tmp/files",
			true,
		)
		assert.NotNil(t, fetcher)
		assert.NotNil(t, fetcher.kubernetesFetcher)
		assert.NotNil(t, fetcher.filesystemFetcher)
	})

	t.Run("creates fetcher with minimal config", func(t *testing.T) {
		fetcher := NewCompositeFetcherWithDefaults(
			nil,
			nil,
			"",
			"",
			false,
		)
		assert.NotNil(t, fetcher)
		assert.Nil(t, fetcher.kubernetesFetcher)
		assert.NotNil(t, fetcher.filesystemFetcher)
	})
}

func TestCompositeFetcher_FetchResources(t *testing.T) {
	t.Run("successfully fetches resources", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
			fetchData: map[string]interface{}{
				"test": "data",
			},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)

		rule := &mockRule{
			ruleType:   scanner.RuleTypeCEL,
			identifier: "test-rule",
			inputs: []scanner.Input{
				&mockInput{
					name:      "test",
					inputType: scanner.InputTypeFile,
					spec:      &mockInputSpec{valid: true},
				},
			},
		}

		result, warnings, err := fetcher.FetchResources(context.Background(), rule, nil)
		require.NoError(t, err)
		assert.Nil(t, warnings)
		assert.Equal(t, "data", result["test"])
	})

	t.Run("returns error on fetch failure", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
			fetchError:     errors.New("fetch failed"),
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)

		rule := &mockRule{
			ruleType:   scanner.RuleTypeCEL,
			identifier: "test-rule",
			inputs: []scanner.Input{
				&mockInput{
					name:      "test",
					inputType: scanner.InputTypeFile,
					spec:      &mockInputSpec{valid: true},
				},
			},
		}

		result, warnings, err := fetcher.FetchResources(context.Background(), rule, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Nil(t, warnings)
		assert.Contains(t, err.Error(), "fetch failed")
	})
}

func TestCompositeFetcher_FetchInputs(t *testing.T) {
	t.Run("successfully fetches multiple input types", func(t *testing.T) {
		fetcher := NewCompositeFetcher()

		// Mock file fetcher
		fileFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
			fetchData: map[string]interface{}{
				"config": "file data",
			},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, fileFetcher)

		// Mock system fetcher
		systemFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeSystem},
			fetchData: map[string]interface{}{
				"nginx": "system data",
			},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeSystem, systemFetcher)

		inputs := []scanner.Input{
			&mockInput{
				name:      "config",
				inputType: scanner.InputTypeFile,
				spec:      &mockInputSpec{valid: true},
			},
			&mockInput{
				name:      "nginx",
				inputType: scanner.InputTypeSystem,
				spec:      &mockInputSpec{valid: true},
			},
		}

		result, err := fetcher.FetchInputs(inputs, nil)
		require.NoError(t, err)
		assert.Equal(t, "file data", result["config"])
		assert.Equal(t, "system data", result["nginx"])
	})

	t.Run("returns error for unsupported input type", func(t *testing.T) {
		fetcher := NewCompositeFetcher()

		inputs := []scanner.Input{
			&mockInput{
				name:      "unsupported",
				inputType: scanner.InputTypeHTTP,
				spec:      &mockInputSpec{valid: true},
			},
		}

		result, err := fetcher.FetchInputs(inputs, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no fetcher available for input type")
	})

	t.Run("returns error on fetcher failure", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
			fetchError:     errors.New("fetcher error"),
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)

		inputs := []scanner.Input{
			&mockInput{
				name:      "test",
				inputType: scanner.InputTypeFile,
				spec:      &mockInputSpec{valid: true},
			},
		}

		result, err := fetcher.FetchInputs(inputs, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to fetch inputs for type")
	})
}

func TestCompositeFetcher_SupportsInputType(t *testing.T) {
	t.Run("supports custom fetcher types", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)

		assert.True(t, fetcher.SupportsInputType(scanner.InputTypeFile))
		assert.False(t, fetcher.SupportsInputType(scanner.InputTypeHTTP))
	})

	t.Run("supports built-in fetcher types", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fetcher.SetFilesystemFetcher(NewFilesystemFetcher(""))
		fetcher.SetKubernetesFetcher(NewKubernetesFetcher(nil, nil))

		assert.True(t, fetcher.SupportsInputType(scanner.InputTypeFile))
		assert.True(t, fetcher.SupportsInputType(scanner.InputTypeKubernetes))
	})
}

func TestCompositeFetcher_GetFetcherForType(t *testing.T) {
	t.Run("returns custom fetcher first", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)
		fetcher.SetFilesystemFetcher(NewFilesystemFetcher(""))

		result := fetcher.getFetcherForType(scanner.InputTypeFile)
		assert.Equal(t, mockFetcher, result)
	})

	t.Run("returns built-in fetcher if no custom", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fileFetcher := NewFilesystemFetcher("")
		fetcher.SetFilesystemFetcher(fileFetcher)

		result := fetcher.getFetcherForType(scanner.InputTypeFile)
		assert.Equal(t, fileFetcher, result)
	})

	t.Run("returns nil for unsupported type", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		result := fetcher.getFetcherForType(scanner.InputTypeHTTP)
		assert.Nil(t, result)
	})
}

func TestCompositeFetcher_RegisterCustomFetcher(t *testing.T) {
	t.Run("registers custom fetcher", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
		}

		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, mockFetcher)
		assert.Equal(t, mockFetcher, fetcher.customFetchers[scanner.InputTypeFile])
	})
}

func TestCompositeFetcher_SetFetchers(t *testing.T) {
	t.Run("sets kubernetes fetcher", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		kubeFetcher := NewKubernetesFetcher(nil, nil)

		fetcher.SetKubernetesFetcher(kubeFetcher)
		assert.Equal(t, kubeFetcher, fetcher.kubernetesFetcher)
	})

	t.Run("sets filesystem fetcher", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fileFetcher := NewFilesystemFetcher("")

		fetcher.SetFilesystemFetcher(fileFetcher)
		assert.Equal(t, fileFetcher, fetcher.filesystemFetcher)
	})

}

func TestCompositeFetcher_GetSupportedInputTypes(t *testing.T) {
	t.Run("returns all supported types", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fetcher.SetFilesystemFetcher(NewFilesystemFetcher(""))

		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeHTTP},
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeHTTP, mockFetcher)

		types := fetcher.GetSupportedInputTypes()
		assert.Contains(t, types, scanner.InputTypeHTTP)
		assert.NotContains(t, types, scanner.InputTypeKubernetes)
	})

	t.Run("returns empty for no fetchers", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		types := fetcher.GetSupportedInputTypes()
		assert.Empty(t, types)
	})
}

func TestCompositeFetcher_ValidateInputs(t *testing.T) {
	t.Run("validates supported inputs", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fetcher.SetFilesystemFetcher(NewFilesystemFetcher(""))

		inputs := []scanner.Input{
			&mockInput{
				name:      "test",
				inputType: scanner.InputTypeFile,
				spec:      &mockInputSpec{valid: true},
			},
		}

		err := fetcher.ValidateInputs(inputs)
		assert.NoError(t, err)
	})

	t.Run("fails for unsupported input type", func(t *testing.T) {
		fetcher := NewCompositeFetcher()

		inputs := []scanner.Input{
			&mockInput{
				name:      "unsupported",
				inputType: scanner.InputTypeHTTP,
				spec:      &mockInputSpec{valid: true},
			},
		}

		err := fetcher.ValidateInputs(inputs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported input type")
	})

	t.Run("fails for invalid input spec", func(t *testing.T) {
		fetcher := NewCompositeFetcher()
		fetcher.SetFilesystemFetcher(NewFilesystemFetcher(""))

		inputs := []scanner.Input{
			&mockInput{
				name:      "test",
				inputType: scanner.InputTypeFile,
				spec:      &mockInputSpec{valid: false},
			},
		}

		err := fetcher.ValidateInputs(inputs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input spec")
	})
}

func TestCompositeFetcherBuilder(t *testing.T) {
	t.Run("builds composite fetcher with builder pattern", func(t *testing.T) {
		builder := NewCompositeFetcherBuilder()
		assert.NotNil(t, builder)
		assert.NotNil(t, builder.fetcher)
	})

	t.Run("builds with kubernetes support", func(t *testing.T) {
		fetcher := NewCompositeFetcherBuilder().
			WithKubernetes(nil, nil).
			Build()

		assert.NotNil(t, fetcher.kubernetesFetcher)
	})

	t.Run("builds with kubernetes file support", func(t *testing.T) {
		fetcher := NewCompositeFetcherBuilder().
			WithKubernetesFiles("/tmp/api-resources").
			Build()

		assert.NotNil(t, fetcher.kubernetesFetcher)
	})

	t.Run("builds with filesystem support", func(t *testing.T) {
		fetcher := NewCompositeFetcherBuilder().
			WithFilesystem("/tmp").
			Build()

		assert.NotNil(t, fetcher.filesystemFetcher)
	})

	t.Run("builds with custom fetcher", func(t *testing.T) {
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeHTTP},
		}

		fetcher := NewCompositeFetcherBuilder().
			WithCustomFetcher(scanner.InputTypeHTTP, mockFetcher).
			Build()

		assert.Equal(t, mockFetcher, fetcher.customFetchers[scanner.InputTypeHTTP])
	})

	t.Run("builds with all components", func(t *testing.T) {
		mockFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeHTTP},
		}

		fetcher := NewCompositeFetcherBuilder().
			WithKubernetesFiles("/tmp/api-resources").
			WithFilesystem("/tmp").
			WithCustomFetcher(scanner.InputTypeHTTP, mockFetcher).
			Build()

		assert.NotNil(t, fetcher.kubernetesFetcher)
		assert.NotNil(t, fetcher.filesystemFetcher)
		assert.Equal(t, mockFetcher, fetcher.customFetchers[scanner.InputTypeHTTP])
	})
}

func TestCompositeFetcher_Integration(t *testing.T) {
	t.Run("integrates with real filesystem fetcher", func(t *testing.T) {
		// Create temporary test file
		tempDir, err := os.MkdirTemp("", "composite_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		fetcher := NewCompositeFetcherBuilder().
			WithFilesystem(tempDir).
			Build()

		input := scanner.NewFileInput("testfile", "test.txt", "text", false, false)
		result, err := fetcher.FetchInputs([]scanner.Input{input}, nil)

		require.NoError(t, err)
		assert.Equal(t, "test content", result["testfile"])
	})
}

func TestCelVariable(t *testing.T) {
	t.Run("implements CelVariable interface", func(t *testing.T) {
		gvk := schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		}

		variable := &CelVariable{
			name:      "test-var",
			namespace: "default",
			value:     "test-value",
			gvk:       gvk,
		}

		assert.Equal(t, "test-var", variable.Name())
		assert.Equal(t, "default", variable.Namespace())
		assert.Equal(t, "test-value", variable.Value())
		assert.Equal(t, gvk, variable.GroupVersionKind())
	})
}

func TestCompositeFetcher_ErrorHandling(t *testing.T) {
	t.Run("handles fetcher panic gracefully", func(t *testing.T) {
		fetcher := NewCompositeFetcher()

		// Mock fetcher that panics
		panicFetcher := &mockInputFetcher{
			supportedTypes: []scanner.InputType{scanner.InputTypeFile},
			fetchError:     errors.New("panic: something went wrong"),
		}
		fetcher.RegisterCustomFetcher(scanner.InputTypeFile, panicFetcher)

		inputs := []scanner.Input{
			&mockInput{
				name:      "test",
				inputType: scanner.InputTypeFile,
				spec:      &mockInputSpec{valid: true},
			},
		}

		result, err := fetcher.FetchInputs(inputs, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "panic: something went wrong")
	})
}

func TestCompositeFetcher_EmptyInputs(t *testing.T) {
	t.Run("handles empty inputs gracefully", func(t *testing.T) {
		fetcher := NewCompositeFetcher()

		result, err := fetcher.FetchInputs([]scanner.Input{}, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
