package event

import "log/slog"

// Default configuration values for the event bus.
const (
	DefaultBufferSize  = 1024
	DefaultWorkerCount = 4
)

// busConfig holds the configuration for the event bus.
type busConfig struct {
	bufferSize   int
	workerCount  int
	errorHandler func(topic string, err error)
	logger       *slog.Logger
}

// defaultConfig returns the default bus configuration.
func defaultConfig() *busConfig {
	return &busConfig{
		bufferSize:  DefaultBufferSize,
		workerCount: DefaultWorkerCount,
		errorHandler: func(topic string, err error) {
			// Default: silent (use WithErrorHandler to customize)
		},
		logger: nil,
	}
}

// Option configures the event bus.
type Option func(*busConfig)

// WithBufferSize sets the size of the event channel buffer.
// A larger buffer allows more events to be queued before blocking publishers.
// Default: 1024
func WithBufferSize(size int) Option {
	return func(c *busConfig) {
		if size > 0 {
			c.bufferSize = size
		}
	}
}

// WithWorkerCount sets the number of worker goroutines processing events.
// More workers allow parallel event processing but increase resource usage.
// Default: 4
func WithWorkerCount(count int) Option {
	return func(c *busConfig) {
		if count > 0 {
			c.workerCount = count
		}
	}
}

// WithErrorHandler sets a callback for handler errors.
// The callback receives the topic and the error returned by the handler.
// Use this for logging, metrics, or error recovery.
func WithErrorHandler(fn func(topic string, err error)) Option {
	return func(c *busConfig) {
		if fn != nil {
			c.errorHandler = fn
		}
	}
}

// WithLogger sets a structured logger for the bus.
// The bus will log lifecycle events and errors at appropriate levels.
func WithLogger(logger *slog.Logger) Option {
	return func(c *busConfig) {
		c.logger = logger
	}
}
