package main

import (
	"fmt"
	"log"
	"os"
)

type AgentConfig struct {
	taskID        string
	taskAuthToken string
	agentEndpoint string
}

func main() {
	// Retrieve task ID from environment variable
	_, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Retrieve task details from the endpoint
	// GET /agent/v1/tasks/{task_id}

	// Send 'start' update to POST /agent/v1/tasks/{task_id}/start

	// Execute the task based on its type

	// Send 'complete' update to POST /agent/v1/tasks/{task_id}/complete
}

func loadConfig() (*AgentConfig, error) {
	config := &AgentConfig{
		taskID:        os.Getenv("TASK_ID"),
		taskAuthToken: os.Getenv("TASK_AUTH_TOKEN"),
		agentEndpoint: os.Getenv("AGENT_ENDPOINT"),
	}

	// Basic validation
	if config.taskID == "" {
		return nil, fmt.Errorf("TASK_ID environment variable is required")
	}
	if config.taskAuthToken == "" {
		return nil, fmt.Errorf("TASK_AUTH_TOKEN environment variable is required")
	}
	if config.agentEndpoint == "" {
		return nil, fmt.Errorf("AGENT_ENDPOINT environment variable is required")
	}

	return config, nil
}
