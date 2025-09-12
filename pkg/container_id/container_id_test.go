package container_id

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetcontainer_id(t *testing.T) {
	tests := []struct {
		name                   string
		ecsMetadataURI         string
		hostname               string
		expectedPrefix         string
		expectedExact          string
		shouldHaveRandomPrefix bool
	}{
		{
			name:                   "no environment variables - should get random ID",
			ecsMetadataURI:         "",
			hostname:               "",
			shouldHaveRandomPrefix: true,
			expectedPrefix:         "random-",
		},
		{
			name:                   "hostname set to localhost - should get random ID",
			ecsMetadataURI:         "",
			hostname:               "localhost",
			shouldHaveRandomPrefix: true,
			expectedPrefix:         "random-",
		},
		{
			name:           "valid hostname - should return hostname",
			ecsMetadataURI: "",
			hostname:       "test-hostname",
			expectedExact:  "test-hostname",
		},
		{
			name:           "another valid hostname - should return hostname",
			ecsMetadataURI: "",
			hostname:       "my-app-container",
			expectedExact:  "my-app-container",
		},
		{
			name:                   "ECS metadata URI set but no hostname - should get random ID",
			ecsMetadataURI:         "http://invalid-ecs-endpoint",
			hostname:               "",
			shouldHaveRandomPrefix: true,
			expectedPrefix:         "random-",
		},
		{
			name:                   "ECS metadata URI set with localhost hostname - should get random ID",
			ecsMetadataURI:         "http://invalid-ecs-endpoint",
			hostname:               "localhost",
			shouldHaveRandomPrefix: true,
			expectedPrefix:         "random-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalHostname := os.Getenv("HOSTNAME")
			originalECS := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")

			// Clean up environment
			defer func() {
				if originalHostname != "" {
					os.Setenv("HOSTNAME", originalHostname)
				} else {
					os.Unsetenv("HOSTNAME")
				}
				if originalECS != "" {
					os.Setenv("ECS_CONTAINER_METADATA_URI_V4", originalECS)
				} else {
					os.Unsetenv("ECS_CONTAINER_METADATA_URI_V4")
				}
			}()

			// Set up test environment
			os.Unsetenv("HOSTNAME")
			os.Unsetenv("ECS_CONTAINER_METADATA_URI_V4")

			if tt.ecsMetadataURI != "" {
				os.Setenv("ECS_CONTAINER_METADATA_URI_V4", tt.ecsMetadataURI)
			}
			if tt.hostname != "" {
				os.Setenv("HOSTNAME", tt.hostname)
			}

			// Execute the function
			container_id := Getcontainer_id()

			// Assertions
			require.NotEmpty(t, container_id, "Container ID should not be empty")

			if tt.shouldHaveRandomPrefix {
				assert.True(t, strings.HasPrefix(container_id, tt.expectedPrefix),
					"Expected container ID to start with '%s', got: %s", tt.expectedPrefix, container_id)
				assert.Equal(t, 43, len(container_id), "Random container ID should be 43 characters long")
			} else if tt.expectedExact != "" {
				assert.Equal(t, tt.expectedExact, container_id,
					"Expected exact container ID match")
			}
		})
	}
}

func TestFetchECScontainer_id(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   func(w http.ResponseWriter, r *http.Request)
		expectedResult string
	}{
		{
			name: "successful ECS metadata fetch",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ECSMetadata{DockerId: "test-container-id"})
			},
			expectedResult: "test-container-id",
		},
		{
			name: "HTTP error response",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedResult: "",
		},
		{
			name: "empty DockerId in response",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ECSMetadata{DockerId: ""})
			},
			expectedResult: "",
		},
		{
			name: "malformed JSON response",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"DockerId": invalid json}`))
			},
			expectedResult: "",
		},
		{
			name: "missing DockerId field",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"SomeOtherField": "value"})
			},
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			result := fetchECScontainer_id(server.URL)
			assert.Equal(t, tt.expectedResult, result)
		})
	}

	// Test invalid URL
	t.Run("invalid URL", func(t *testing.T) {
		result := fetchECScontainer_id("invalid-url")
		assert.Empty(t, result)
	})
}

func TestGenerateRandomcontainer_id(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "generates random ID with correct format"},
		{name: "generates unique IDs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generateRandomcontainer_id()

			assert.True(t, strings.HasPrefix(id, "random-"),
				"Expected random- prefix, got %s", id)
			assert.Equal(t, 43, len(id),
				"Expected 43 characters, got %d", len(id))

			// Test uniqueness by generating another ID
			id2 := generateRandomcontainer_id()
			assert.NotEqual(t, id, id2,
				"Expected unique IDs, got duplicate: %s", id)
		})
	}
}

func TestGetcontainer_id_ECSIntegration(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   func(w http.ResponseWriter, r *http.Request)
		hostname       string
		expectedResult string
	}{
		{
			name: "ECS success - should use ECS even if hostname is set",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ECSMetadata{DockerId: "ecs-container-123"})
			},
			hostname:       "should-be-ignored",
			expectedResult: "ecs-container-123",
		},
		{
			name: "ECS failure - should fall back to hostname",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			hostname:       "fallback-hostname",
			expectedResult: "fallback-hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalECS := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
			originalHostname := os.Getenv("HOSTNAME")

			defer func() {
				if originalECS != "" {
					os.Setenv("ECS_CONTAINER_METADATA_URI_V4", originalECS)
				} else {
					os.Unsetenv("ECS_CONTAINER_METADATA_URI_V4")
				}
				if originalHostname != "" {
					os.Setenv("HOSTNAME", originalHostname)
				} else {
					os.Unsetenv("HOSTNAME")
				}
			}()

			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			os.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)
			os.Setenv("HOSTNAME", tt.hostname)

			id := Getcontainer_id()
			assert.Equal(t, tt.expectedResult, id)
		})
	}
}
