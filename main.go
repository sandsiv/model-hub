package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"model-hub/api"
	"model-hub/config"
	"model-hub/helper"
	"model-hub/workers"
	"os"
)

func main() {
	cfg, err := config.Load(helper.GetEnv("CONFIG_PATH", "config.yaml"))
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := createLogger()

	defer logger.Sync()

	workerManager := workers.NewWorkerManager(cfg, logger)
	logger.Info("Starting workers")
	go workerManager.Initialize()

	api.NewAPIServer(workerManager, logger)
}

func createLogger() *zap.Logger {
	// info level enabler
	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.InfoLevel || level == zapcore.DebugLevel || level == zapcore.WarnLevel
	})

	// error and fatal level enabler
	errorFatalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.ErrorLevel || level == zapcore.FatalLevel
	})

	// write syncers
	stdoutSyncer := zapcore.Lock(os.Stdout)
	stderrSyncer := zapcore.Lock(os.Stderr)

	// create a custom encoder configuration without time
	customEncoderConfig := zap.NewDevelopmentEncoderConfig()
	customEncoderConfig.TimeKey = ""

	// tee core
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(customEncoderConfig),
			stdoutSyncer,
			infoLevel,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(customEncoderConfig),
			stderrSyncer,
			errorFatalLevel,
		),
	)

	return zap.New(core)
}
