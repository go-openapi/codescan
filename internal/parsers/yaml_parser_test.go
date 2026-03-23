package parsers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

var errSetterFailed = errors.New("setter failed")

func TestYamlParser(t *testing.T) {
	t.Parallel()

	setter := func(out *string, called *int) func(json.RawMessage) error {
		return func(in json.RawMessage) error {
			*called++
			*out = string(in)

			return nil
		}
	}

	t.Run("with happy path", func(t *testing.T) {
		t.Run("should parse security definitions object as YAML", func(t *testing.T) {
			setterCalled := 0
			var actualJSON string
			parser := NewYAMLParser(WithMatcher(rxSecurity), WithSetter(setter(&actualJSON, &setterCalled)))

			lines := []string{
				"SecurityDefinitions:",
				"  api_key:",
				"    type: apiKey",
				"    name: X-API-KEY",
				"  petstore_auth:",
				"    type: oauth2",
				"    scopes:",
				"      'write:pets': modify pets in your account",
				"      'read:pets': read your pets",
			}

			require.TrueT(t, parser.Matches(lines[0]))
			require.NoError(t, parser.Parse(lines))
			require.EqualT(t, 1, setterCalled)

			const expectedJSON = `{"SecurityDefinitions":{"api_key":{"name":"X-API-KEY","type":"apiKey"},` +
				`"petstore_auth":{"scopes":{"read:pets":"read your pets","write:pets":"modify pets in your account"},"type":"oauth2"}}}`

			require.JSONEqT(t, expectedJSON, actualJSON)
		})
	})

	t.Run("with edge cases", func(t *testing.T) {
		t.Run("should handle empty input", func(t *testing.T) {
			setterCalled := 0
			var actualJSON string
			parser := NewYAMLParser(WithMatcher(rxSecurity), WithSetter(setter(&actualJSON, &setterCalled)))

			require.FalseT(t, parser.Matches(""))
			require.NoError(t, parser.Parse([]string{}))
			require.Zero(t, setterCalled)
		})

		t.Run("should handle nil input", func(t *testing.T) {
			setterCalled := 0
			var actualJSON string
			parser := NewYAMLParser(WithMatcher(rxSecurity), WithSetter(setter(&actualJSON, &setterCalled)))

			require.NoError(t, parser.Parse(nil))
			require.Zero(t, setterCalled)
		})

		t.Run("should handle bad indentation", func(t *testing.T) {
			setterCalled := 0
			var actualJSON string
			parser := NewYAMLParser(WithMatcher(rxSecurity), WithSetter(setter(&actualJSON, &setterCalled)))
			lines := []string{
				"SecurityDefinitions:",
				"\t\tapi_key:",
				"  type: apiKey",
			}

			require.TrueT(t, parser.Matches(lines[0]))
			err := parser.Parse(lines)
			require.Error(t, err)
			require.StringContainsT(t, err.Error(), "yaml: line 2:")
			require.Zero(t, setterCalled)
		})

		t.Run("should catch YAML errors", func(t *testing.T) {
			setterCalled := 0
			var actualJSON string
			parser := NewYAMLParser(WithMatcher(rxSecurity), WithSetter(setter(&actualJSON, &setterCalled)))
			lines := []string{
				"SecurityDefinitions:",
				"  api_key",
				"    type: apiKey",
			}

			require.TrueT(t, parser.Matches(lines[0]))
			err := parser.Parse(lines)
			require.Error(t, err)
			require.StringContainsT(t, err.Error(), "yaml: line 3: mapping value")
			require.Zero(t, setterCalled)
		})

		t.Run("should handle nil rx in Matches", func(t *testing.T) {
			parser := NewYAMLParser(WithSetter(func(_ json.RawMessage) error { return nil }))
			require.FalseT(t, parser.Matches("anything"))
		})

		t.Run("should handle nil setter", func(t *testing.T) {
			parser := NewYAMLParser(WithMatcher(rxSecurity))
			lines := []string{
				"SecurityDefinitions:",
				"  api_key:",
				"    type: apiKey",
			}
			require.NoError(t, parser.Parse(lines))
		})

		t.Run("should propagate setter error", func(t *testing.T) {
			parser := NewYAMLParser(
				WithMatcher(rxSecurity),
				WithSetter(func(_ json.RawMessage) error { return errSetterFailed }),
			)
			lines := []string{
				"SecurityDefinitions:",
				"  api_key:",
				"    type: apiKey",
			}
			err := parser.Parse(lines)
			require.Error(t, err)
			require.ErrorIs(t, err, errSetterFailed)
		})
	})
}
