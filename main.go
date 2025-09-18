package main

import (
	"fmt"
	"log"
)

type Application struct {
	config    *Config
	apiClient *APIClient
	display   *DisplayManager
	scheduler *Scheduler
}

func NewApplication() *Application {
	return &Application{}
}

func (app *Application) Initialize() error {
	configManager := NewConfigManager()
	config, err := configManager.LoadConfig()
	if err != nil {
		configManager.printUsage()
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	app.config = config

	app.apiClient = NewAPIClient(config)

	app.display = NewDisplayManager(config)

	app.scheduler = NewScheduler(config, app.apiClient, app.display)

	return nil
}

func (app *Application) Run() error {
	if err := app.scheduler.TestInitialConnection(); err != nil {
		if app.display != nil {
			app.display.RestoreTerminal()
		}
		return fmt.Errorf("initial connection test failed: %w", err)
	}

	return app.scheduler.Start()
}

func (app *Application) Shutdown() {
	if app.scheduler != nil {
		app.scheduler.Stop()
	}
	if app.display != nil {
		app.display.RestoreTerminal()
	}
}

func main() {
	app := NewApplication()

	if err := app.Initialize(); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	defer func() {
		app.Shutdown()

		if app.display != nil {
			app.display.RestoreTerminal()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
