package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Scheduler struct {
	config       *Config
	apiClient    *APIClient
	display      *DisplayManager
	ctx          context.Context
	cancel       context.CancelFunc
	ticker       *time.Ticker
	running      bool
	dataChannel  chan *APIResponse
	errorChannel chan error
}

func NewScheduler(config *Config, apiClient *APIClient, display *DisplayManager) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		config:       config,
		apiClient:    apiClient,
		display:      display,
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
		dataChannel:  make(chan *APIResponse, 1),
		errorChannel: make(chan error, 1),
	}
}

func (s *Scheduler) Start() error {
	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.display.StartFullScreenMode()

	s.running = true
	s.ticker = time.NewTicker(s.config.PollInterval)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go s.fetchData()

	for {
		select {
		case <-s.ctx.Done():

			s.cleanup()
			return nil

		case <-signalChan:

			s.display.RestoreTerminal()
			s.Stop()
			return nil

		case <-s.ticker.C:

			go s.fetchData()

		case response := <-s.dataChannel:

			grouped := GroupDevicesByLogicalDevice(response)
			s.display.UpdateTerminalSize()
			s.display.Render(grouped, nil)

		case err := <-s.errorChannel:

			s.display.Render(nil, err)
		}
	}
}

func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.cancel()
}

func (s *Scheduler) fetchData() {
	select {
	case <-s.ctx.Done():
		return
	default:
		response, err := s.apiClient.FetchDevicesWithRetry(2)
		if err != nil {
			select {
			case s.errorChannel <- err:
			case <-s.ctx.Done():
			}
		} else {
			select {
			case s.dataChannel <- response:
			case <-s.ctx.Done():
			}
		}
	}
}

func (s *Scheduler) cleanup() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.running = false

	close(s.dataChannel)
	close(s.errorChannel)
}

func (s *Scheduler) UpdateConfig(config *Config) {
	s.config = config

	if s.running && s.ticker != nil {
		s.ticker.Stop()
		s.ticker = time.NewTicker(config.PollInterval)
	}
}

func (s *Scheduler) IsRunning() bool {
	return s.running
}

func (s *Scheduler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running":       s.running,
		"poll_interval": s.config.PollInterval,
		"api_endpoint":  s.config.APIEndpoint,
	}
}

func (s *Scheduler) TestInitialConnection() error {
	err := s.apiClient.Login(s.config.Username, s.config.Password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	err = s.apiClient.TestConnection()
	if err != nil {
		return fmt.Errorf("initial connection test failed: %w", err)
	}

	return nil
}

func (s *Scheduler) RunOnce() error {
	response, err := s.apiClient.FetchDevicesWithRetry(2)
	if err != nil {
		s.display.Render(nil, err)
		return err
	}

	grouped := GroupDevicesByLogicalDevice(response)
	s.display.Render(grouped, nil)
	return nil
}
