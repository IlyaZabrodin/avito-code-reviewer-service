package bootstrap

import (
	"CodeRewievService/internal/controllers"
	"CodeRewievService/internal/database"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultShutdownTimeout = 10 * time.Second
	goroutineWaitTimeout   = 5 * time.Second
)

type Application struct {
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *slog.Logger
	dependencies   *Dependencies
	shutdownSignal chan os.Signal
	state          *applicationState
}

type Dependencies struct {
	server Server
}

type applicationState struct {
	mu        sync.RWMutex
	isRunning bool
	wg        sync.WaitGroup
}

func InitApplication() *Application {
	ctx, cancel := context.WithCancel(context.Background())
	return &Application{
		ctx:            ctx,
		cancel:         cancel,
		logger:         slog.Default(),
		dependencies:   &Dependencies{},
		shutdownSignal: make(chan os.Signal, 1),
		state: &applicationState{
			isRunning: false,
		},
	}
}

func (app *Application) Run() error {
	app.initializeDependencies()

	if err := app.start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	app.logger.Info("Application started successfully")
	app.setupShutdownHandlers()

	if err := app.waitForShutdown(); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	app.logger.Info("Application shutdown completed successfully")
	return nil
}

func (app *Application) initializeDependencies() {
	app.logger.Info("Initializing application dependencies...")

	if err := godotenv.Load(); err != nil {
		app.logger.Warn("Failed to load .env file, using environment variables", "error", err)
	}

	db := database.InitializeConnection()
	config := app.loadServerConfig()

	app.dependencies.server = controllers.NewHTTPServer(app.logger, db, config.address, config.port)
	app.logger.Info("Dependencies initialized successfully", "address", config.address, "port", config.port)
}

type serverConfig struct {
	address string
	port    int
}

func (app *Application) loadServerConfig() serverConfig {
	address := os.Getenv("APP_ADDRESS")
	if address == "" {
		address = "0.0.0.0"
	}

	portStr := os.Getenv("APP_PORT")
	port := 8080
	if portStr != "" {
		if parsedPort, err := parseInt(portStr); err == nil {
			port = parsedPort
		} else {
			app.logger.Warn("Invalid APP_PORT value, using default", "port", port, "error", err)
		}
	}

	return serverConfig{
		address: address,
		port:    port,
	}
}

func (app *Application) start() error {
	app.logger.Info("Starting application components...")

	app.state.mu.Lock()
	if app.state.isRunning {
		app.state.mu.Unlock()
		return fmt.Errorf("application is already running")
	}
	app.state.mu.Unlock()

	app.state.wg.Add(1)
	go app.runServer()

	app.setStateRunning(true)
	app.logger.Info("Application components started successfully")

	return nil
}

func (app *Application) runServer() {
	defer app.state.wg.Done()

	if err := app.dependencies.server.Start(app.ctx); err != nil {
		app.logger.Error("Server stopped with error", "error", err)
		app.setStateRunning(false)
	}
}

func (app *Application) stop() error {
	app.state.mu.Lock()
	defer app.state.mu.Unlock()

	if !app.state.isRunning {
		app.logger.Debug("Application is not running, nothing to stop")
		return nil
	}

	app.logger.Info("Initiating graceful shutdown...")
	app.cancel()

	if err := app.stopComponents(); err != nil {
		app.logger.Error("Error stopping components", "error", err)
	}

	app.waitForGoroutines()
	app.setStateRunning(false)

	close(app.shutdownSignal)
	app.logger.Info("Application stopped successfully")

	return nil
}

func (app *Application) stopComponents() error {
	app.logger.Debug("Stopping application components...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := app.dependencies.server.Stop(shutdownCtx, defaultShutdownTimeout); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}

func (app *Application) waitForGoroutines() {
	app.logger.Debug("Waiting for goroutines to finish...")

	done := make(chan struct{})
	go func() {
		app.state.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.logger.Debug("All goroutines finished successfully")
	case <-time.After(goroutineWaitTimeout):
		app.logger.Warn("Goroutine wait timeout exceeded, forcing stop")
	}
}

func (app *Application) setupShutdownHandlers() {
	signal.Notify(app.shutdownSignal, os.Interrupt, syscall.SIGTERM)
}

func (app *Application) waitForShutdown() error {
	select {
	case <-app.ctx.Done():
		app.logger.Info("Context cancelled, initiating shutdown")
		return app.stop()

	case sig := <-app.shutdownSignal:
		app.logger.Info("Received shutdown signal", "signal", sig)
		return app.stop()
	}
}

func (app *Application) setStateRunning(running bool) {
	app.state.mu.Lock()
	defer app.state.mu.Unlock()
	app.state.isRunning = running
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("invalid integer format: %w", err)
	}
	return result, nil
}
