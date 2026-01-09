package log

import (
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
)

type Logger = *slog.Logger

var level slog.Level

func Init() {
	level = getLevel("LOG_LEVEL")
}

func GetLogger(name ...string) Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

func getLevel(key string) slog.Level {
	level := os.Getenv(key)
	if level == "" {
		return slog.LevelInfo
	}
	var lvl slog.Level
	err := json.Unmarshal([]byte(strconv.Quote(level)), &lvl)
	if err != nil {
		return slog.LevelInfo
	}
	return lvl
}
