// Package container_id provides automatic container ID detection for metrics labeling
package container_id

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// ECSMetadata represents the structure of ECS container metadata
type ECSMetadata struct {
	DockerId string `json:"DockerId"`
}

// Getcontainer_id returns the container ID using multiple detection methods in priority order:
// 1. ECS_CONTAINER_METADATA_URI_V4 (AWS ECS Fargate/EC2) https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
// 2. HOSTNAME environment variable (Kubernetes pods)
// 3. Random UUID with "random-" prefix (fallback)
func Getcontainer_id() string {
	// Try ECS metadata endpoint v4 (only modern ECS environments)
	if metadataURI := os.Getenv("ECS_CONTAINER_METADATA_URI_V4"); metadataURI != "" {
		if container_id := fetchECScontainer_id(metadataURI); container_id != "" {
			return container_id
		}
	}

	// Try hostname (Kubernetes pods often have meaningful hostnames)
	if hostname := os.Getenv("HOSTNAME"); hostname != "" && hostname != "localhost" {
		return hostname
	}

	// Final fallback: generate random UUID with prefix
	return generateRandomcontainer_id()
}

// fetchECScontainer_id fetches container ID from ECS metadata endpoint
func fetchECScontainer_id(metadataURI string) string {
	client := &http.Client{
		Timeout: 2 * time.Second, // Short timeout to avoid blocking application startup
	}

	resp, err := client.Get(metadataURI)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var metadata ECSMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return ""
	}

	return metadata.DockerId
}

// generateRandomcontainer_id creates a random UUID with "random-" prefix
func generateRandomcontainer_id() string {
	return fmt.Sprintf("random-%s", uuid.New().String())
}
