package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dezemandje/aule/internal/agent/client"
	"github.com/dezemandje/aule/internal/agent/llm"
	"github.com/dezemandje/aule/internal/agent/runner"
	"github.com/dezemandje/aule/internal/agent/tool"
	"github.com/dezemandje/aule/internal/log"
	"github.com/dezemandje/aule/internal/service/agentapi"
)

var logger log.Logger

// AgentConfig holds the agent configuration
type AgentConfig struct {
	// Task configuration (from environment)
	TaskID        string
	TaskAuthToken string
	AgentEndpoint string

	// LLM configuration
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string

	// Execution configuration
	WorkDir       string
	MaxIterations int
	Standalone    bool // Run without backend (for testing)
}

func main() {
	log.Init()
	logger = log.GetLogger("cmd", "agent")

	// Setup signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	// Run the agent
	if err := runAgent(ctx, cfg); err != nil {
		logger.Error("Agent failed", "err", err)
		os.Exit(1)
	}

	logger.Info("Agent completed successfully")
}

func runAgent(ctx context.Context, cfg *AgentConfig) error {
	// Create LLM provider
	llmProvider := llm.NewOpenAIProvider(llm.OpenAIConfig{
		BaseURL: cfg.OpenAIBaseURL,
		APIKey:  cfg.OpenAIAPIKey,
		Model:   cfg.OpenAIModel,
	})

	// Create tool registry
	tools := tool.DefaultRegistry()

	// Create runner
	agentRunner := runner.NewRunner(runner.Config{
		Provider:      llmProvider,
		Tools:         tools,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.OpenAIModel,
		Temperature:   0.0,
		MaxTokens:     4096,
	})

	// Standalone mode (for testing without backend)
	if cfg.Standalone {
		return runStandalone(ctx, cfg, agentRunner)
	}

	// Normal mode: connect to backend
	return runWithBackend(ctx, cfg, agentRunner)
}

func runStandalone(ctx context.Context, cfg *AgentConfig, agentRunner *runner.Runner) error {
	logger.Info("Running in standalone mode")

	// Read task prompt from stdin or use default
	taskPrompt := "Explore this directory and describe what you find. List the main files and their purposes."

	// Check for prompt in environment
	if prompt := os.Getenv("AGENT_PROMPT"); prompt != "" {
		taskPrompt = prompt
	}

	// Set up callbacks for progress reporting
	agentRunner.SetCallbacks(
		func(name string, input json.RawMessage) {
			logger.Info("Tool call", "name", name)
		},
		func(name string, output string, isError bool) {
			if isError {
				logger.Warn("Tool error", "name", name, "error", output)
			} else {
				logger.Debug("Tool result", "name", name, "outputLen", len(output))
			}
		},
		func(text string) {
			// Print agent's text output
			fmt.Println(text)
		},
		func(iteration int, usage llm.TokenUsage) {
			logger.Debug("Iteration complete",
				"iteration", iteration,
				"inputTokens", usage.InputTokens,
				"outputTokens", usage.OutputTokens,
			)
		},
	)

	// Run the agent
	result, err := agentRunner.Run(ctx, &runner.TaskInput{
		SystemPrompt: runner.BuildSystemPrompt("exploration", "Standalone Task", ""),
		UserPrompt:   taskPrompt,
		WorkDir:      cfg.WorkDir,
		AllowedTools: []string{"read", "glob", "grep", "bash"},
	})

	if err != nil {
		logger.Error("Agent run failed", "err", err)
	}

	logger.Info("Agent run completed",
		"success", result.Success,
		"iterations", result.Iterations,
		"toolCalls", len(result.ToolCalls),
		"inputTokens", result.TokensUsed.InputTokens,
		"outputTokens", result.TokensUsed.OutputTokens,
	)

	if result.Error != "" {
		return fmt.Errorf("agent error: %s", result.Error)
	}

	return nil
}

func runWithBackend(ctx context.Context, cfg *AgentConfig, agentRunner *runner.Runner) error {
	logger.Info("Running with backend", "endpoint", cfg.AgentEndpoint, "taskID", cfg.TaskID)

	// Create backend client
	backendClient := client.NewClient(cfg.AgentEndpoint, cfg.TaskAuthToken)

	// 1. Fetch task details
	logger.Info("Fetching task details")
	taskDetails, err := backendClient.GetTask(ctx, cfg.TaskID)
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	logger.Info("Task fetched",
		"title", taskDetails.Task.Title,
		"type", taskDetails.Task.Type,
		"tools", taskDetails.AllowedTools,
	)

	// 2. Start the task
	logger.Info("Starting task")
	startResp, err := backendClient.StartTask(ctx, cfg.TaskID)
	if err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	instanceID := startResp.InstanceID
	logger.Info("Task started", "instanceID", instanceID, "sessionID", startResp.SessionID)

	// 3. Set up callbacks
	agentRunner.SetCallbacks(
		func(name string, input json.RawMessage) {
			logger.Info("Tool call", "name", name)
			// Send update to backend
			_ = backendClient.UpdateTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskUpdateRequest{
				ToolCalls: []agentapi.ToolCall{
					{Name: name, Input: string(input)},
				},
			})
		},
		func(name string, output string, isError bool) {
			errStr := ""
			if isError {
				errStr = output
				logger.Warn("Tool error", "name", name, "error", output)
			} else {
				logger.Debug("Tool result", "name", name, "outputLen", len(output))
			}
			// Send update to backend
			_ = backendClient.UpdateTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskUpdateRequest{
				ToolCalls: []agentapi.ToolCall{
					{Name: name, Output: output, Error: errStr},
				},
			})
		},
		func(text string) {
			fmt.Println(text)
			// Send content update
			_ = backendClient.UpdateTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskUpdateRequest{
				Content: text,
			})
		},
		func(iteration int, usage llm.TokenUsage) {
			logger.Debug("Iteration complete",
				"iteration", iteration,
				"inputTokens", usage.InputTokens,
				"outputTokens", usage.OutputTokens,
			)
			// Send token usage update
			_ = backendClient.UpdateTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskUpdateRequest{
				TokensUsed: &agentapi.TokenUsage{
					InputTokens:  usage.InputTokens,
					OutputTokens: usage.OutputTokens,
				},
			})
		},
	)

	// 4. Run the agent
	workDir := taskDetails.WorkDir
	if workDir == "" {
		workDir = cfg.WorkDir
	}

	result, err := agentRunner.Run(ctx, &runner.TaskInput{
		SystemPrompt: taskDetails.SystemPrompt,
		UserPrompt:   taskDetails.Task.Description,
		Context:      taskDetails.Context,
		WorkDir:      workDir,
		AllowedTools: taskDetails.AllowedTools,
	})

	// 5. Report result
	if result.Success {
		logger.Info("Task completed successfully")
		err = backendClient.CompleteTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskCompleteRequest{
			Result: result.Result,
			TokensUsed: agentapi.TokenUsage{
				InputTokens:  result.TokensUsed.InputTokens,
				OutputTokens: result.TokensUsed.OutputTokens,
			},
		})
		if err != nil {
			logger.Warn("Failed to report completion", "err", err)
		}
	} else {
		logger.Error("Task failed", "error", result.Error)
		err = backendClient.FailTask(ctx, cfg.TaskID, instanceID, &agentapi.TaskFailRequest{
			Error: result.Error,
			TokensUsed: agentapi.TokenUsage{
				InputTokens:  result.TokensUsed.InputTokens,
				OutputTokens: result.TokensUsed.OutputTokens,
			},
		})
		if err != nil {
			logger.Warn("Failed to report failure", "err", err)
		}
		return fmt.Errorf("task failed: %s", result.Error)
	}

	return nil
}

func loadConfig() (*AgentConfig, error) {
	config := &AgentConfig{
		// Task config
		TaskID:        os.Getenv("TASK_ID"),
		TaskAuthToken: os.Getenv("TASK_AUTH_TOKEN"),
		AgentEndpoint: os.Getenv("AGENT_ENDPOINT"),

		// LLM config
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
		OpenAIModel:   os.Getenv("OPENAI_MODEL"),

		// Execution config
		WorkDir:    os.Getenv("WORK_DIR"),
		Standalone: os.Getenv("STANDALONE") == "true" || os.Getenv("STANDALONE") == "1",
	}

	// Set defaults
	if config.OpenAIModel == "" {
		config.OpenAIModel = "gpt-4o"
	}
	if config.WorkDir == "" {
		var err error
		config.WorkDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}
	if config.MaxIterations == 0 {
		config.MaxIterations = 50
	}

	// Validate required config
	if config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	// If not standalone, require backend config
	if !config.Standalone {
		if config.TaskID == "" {
			// Default to standalone mode if no task ID
			logger.Info("No TASK_ID provided, running in standalone mode")
			config.Standalone = true
		} else {
			if config.AgentEndpoint == "" {
				return nil, fmt.Errorf("AGENT_ENDPOINT environment variable is required when TASK_ID is set")
			}
		}
	}

	return config, nil
}
