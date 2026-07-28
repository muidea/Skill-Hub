package net

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"log/slog"
)

// HTTPRequestConfig holds configuration for an HTTP request.
type HTTPRequestConfig struct {
	Method      string
	URL         string
	Body        io.Reader
	ContentType string
	Headers     []url.Values
}

// executeHTTPRequest executes an HTTP request with the given configuration.
// It handles common patterns like error logging, response reading, and JSON unmarshaling.
func executeHTTPRequest(httpClient *http.Client, config *HTTPRequestConfig, result any) ([]byte, error) {
	// Create request
	request, err := http.NewRequest(config.Method, config.URL, config.Body)
	if err != nil {
		slog.Error("construct request failed", "url", config.URL, "method", config.Method, "error", err)
		return nil, err
	}

	// Set content type if provided
	if config.ContentType != "" {
		request.Header.Set("content-type", config.ContentType)
	}

	// Set headers
	for _, val := range config.Headers {
		for k, v := range val {
			request.Header.Set(k, v[0])
		}
	}

	// Execute request
	response, err := httpClient.Do(request)
	if err != nil {
		slog.Error(fmt.Sprintf("%s request failed", config.Method), "url", config.URL, "error", err)
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	// Check status code
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected statusCode: %d", response.StatusCode)
	}

	// Read response
	content, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("read response data failed", "url", config.URL, "error", err)
		return nil, err
	}

	// Unmarshal JSON if result is provided
	if result != nil {
		err = json.Unmarshal(content, result)
		if err != nil {
			slog.Error("unmarshal data failed", "url", config.URL, "error", err)
			return nil, err
		}
	}

	return content, nil
}

// executeHTTPRequestWithBody is a convenience wrapper for requests with JSON bodies.
func executeHTTPRequestWithBody(httpClient *http.Client, method, url string, param, result any, headers ...url.Values) ([]byte, error) {
	var body io.Reader
	if param != nil {
		data, err := json.Marshal(param)
		if err != nil {
			slog.Error("marshal param failed", "url", url, "error", err)
			return nil, err
		}
		body = bytes.NewBuffer(data)
	}

	config := &HTTPRequestConfig{
		Method:      method,
		URL:         url,
		Body:        body,
		ContentType: "application/json",
		Headers:     headers,
	}

	return executeHTTPRequest(httpClient, config, result)
}
