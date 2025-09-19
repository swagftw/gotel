package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/GetSimpl/gotel/pkg/logger"
)

// ECSMetadata represents the structure of ECS container metadata
type ECSMetadata struct {
	DockerId string `json:"DockerId"`
}

var client = &http.Client{
	Timeout: 2 * time.Second, // Short timeout to avoid blocking application startup
}

// GetContainerID returns the container ID using multiple detection methods in priority order:
// 1. ECS_CONTAINER_METADATA_URI_V4 (AWS ECS Fargate/EC2) https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
// 2. HOSTNAME environment variable (Kubernetes pods)
// 3. Random UUID with "random-" prefix (fallback)
func GetContainerID() string {
	// Try ECS metadata endpoint v4 (only modern ECS environments)
	if metadataURI := os.Getenv("ECS_CONTAINER_METADATA_URI_V4"); metadataURI != "" {
		if containerID := fetchECSContainerID(metadataURI); containerID != "" {
			return containerID
		}
	}

	// Try hostname (Kubernetes pods often have meaningful hostnames)
	if hostname := os.Getenv("HOSTNAME"); hostname != "" && hostname != "localhost" {
		return hostname
	}

	// Final fallback: generate random UUID with prefix
	return generateRandomContainerID()
}

// fetchECSContainerID fetches container ID from ECS metadata endpoint
func fetchECSContainerID(metadataURI string) string {
	resp, err := client.Get(metadataURI)
	if err != nil {
		return ""
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.Logger.Error("error closing ECS metadata response body", "err", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	metadata := new(ECSMetadata)

	if err = json.NewDecoder(resp.Body).Decode(metadata); err != nil {
		logger.Logger.Error("error decoding ECS metadata response", "err", err.Error())
		return ""
	}

	return metadata.DockerId
}

// generateRandomContainerID creates a random UUID with "random-" prefix
func generateRandomContainerID() string {
	return fmt.Sprintf("random-%s", uuid.New().String())
}
