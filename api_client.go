package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

type APIClient struct {
	client          *http.Client
	config          *Config
	base_url        string
	devicesEndpoint string
	loginEndpoint   string
	authCookie      *http.Cookie
	authenticated   bool
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LimitData struct {
	Limit int32 `json:"limit"`
}

type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %d %s (endpoint: %s)", e.StatusCode, e.Message, e.Endpoint)
}

func NewAPIClient(config *Config) *APIClient {
	cookieJar, _ := cookiejar.New(nil)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   config.RequestTimeout,
		Transport: transport,
		Jar:       cookieJar,
	}

	loginEndpoint := config.BaseURL + "Login"
	devicesEndpoint := config.BaseURL + "ListPhysicalDevices"

	return &APIClient{
		client:          client,
		config:          config,
		loginEndpoint:   loginEndpoint,
		devicesEndpoint: devicesEndpoint,
		authenticated:   false,
	}
}

func (ac *APIClient) Login(login, password string) error {
	loginReq := LoginRequest{
		Login:    login,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", ac.loginEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-api-monitor/1.0")

	resp, err := ac.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Endpoint:   ac.loginEndpoint,
		}
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "Authorization" || cookie.Name == "Autorization" {
			ac.authCookie = cookie
			ac.authenticated = true
			break
		}
	}

	if !ac.authenticated {
		return fmt.Errorf("no Authorization cookie received from login response")
	}

	return nil
}

func (ac *APIClient) FetchDevices() (*APIResponse, error) {
	limitata := LimitData{Limit: 10000}
	jsonData, err := json.Marshal(limitata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal devices request: %w", err)
	}

	if !ac.authenticated {
		return nil, fmt.Errorf("not authenticated - please login first")
	}

	response, err := ac.makeDevicesRequest(jsonData)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusUnauthorized {
			ac.authenticated = false

			if reAuthErr := ac.Login(ac.config.Username, ac.config.Password); reAuthErr != nil {
				return nil, fmt.Errorf("failed to re-authenticate: %w", reAuthErr)
			}

			response, err = ac.makeDevicesRequest(jsonData)
			if err != nil {
				return nil, fmt.Errorf("failed after re-authentication: %w", err)
			}
		} else {
			return nil, err
		}
	}

	return response, nil
}

func (ac *APIClient) makeDevicesRequest(jsonData []byte) (*APIResponse, error) {
	req, err := http.NewRequest("POST", ac.devicesEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-api-monitor/1.0")

	if ac.authCookie != nil {
		req.AddCookie(ac.authCookie)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "authentication expired",
			Endpoint:   ac.devicesEndpoint,
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Endpoint:   ac.devicesEndpoint,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &apiResponse, nil
}

func (ac *APIClient) FetchDevicesWithRetry(maxRetries int) (*APIResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt) * time.Second
			time.Sleep(waitTime)
		}

		response, err := ac.FetchDevices()
		if err == nil {
			return response, nil
		}
		lastErr = err

		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

func (ac *APIClient) TestConnection() error {
	limitata := LimitData{Limit: 10000}
	jsonData, err := json.Marshal(limitata)
	if err != nil {
		return fmt.Errorf("failed to marshal devices request: %w", err)
	}

	err = ac.makeTestRequest(jsonData)
	if err != nil {

		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusUnauthorized {
			ac.authenticated = false

			if reAuthErr := ac.Login(ac.config.Username, ac.config.Password); reAuthErr != nil {
				return fmt.Errorf("failed to re-authenticate during test: %w", reAuthErr)
			}

			err = ac.makeTestRequest(jsonData)
			if err != nil {
				return fmt.Errorf("test failed after re-authentication: %w", err)
			}
		} else {
			return err
		}
	}

	return nil
}

func (ac *APIClient) makeTestRequest(jsonData []byte) error {
	req, err := http.NewRequest("POST", ac.devicesEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-api-monitor/1.0")

	if ac.authCookie != nil {
		req.AddCookie(ac.authCookie)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		fmt.Printf("\n\nerr: %v\n\n", err)
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "authentication expired",
			Endpoint:   ac.devicesEndpoint,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
			Endpoint:   ac.devicesEndpoint,
		}
	}

	return nil
}

func (ac *APIClient) GetEndpoint() string {
	return ac.devicesEndpoint
}

func (ac *APIClient) UpdateConfig(config *Config) {
	ac.config = config
	ac.base_url = config.BaseURL

	ac.client.Timeout = config.RequestTimeout

	transport := ac.client.Transport.(*http.Transport)

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = true

}

func (ac *APIClient) IsAuthenticated() bool {
	return ac.authenticated
}

func (ac *APIClient) Logout() {
	ac.authenticated = false
	ac.authCookie = nil
}

func (ac *APIClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"endpoint":      ac.devicesEndpoint,
		"timeout":       ac.config.RequestTimeout,
		"authenticated": ac.authenticated,
	}
}
