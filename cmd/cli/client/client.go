package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

var ErrUnauthorized = errors.New("session expired, please login again")

type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func New(baseURL, token string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewFromConfig() *APIClient {
	return New(viper.GetString("server_url"), viper.GetString("token"))
}

func (c *APIClient) Do(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if msg, ok := errResp["error"].(string); ok && msg != "" {
				return nil, fmt.Errorf("%s", msg)
			}
		}
		return nil, ErrUnauthorized
	}

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if msg, ok := errResp["error"]; ok {
				return nil, fmt.Errorf("API error (%d): %v", resp.StatusCode, msg)
			}
			if msg, ok := errResp["message"]; ok {
				return nil, fmt.Errorf("API error (%d): %v", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *APIClient) Get(path string) ([]byte, error) {
	return c.Do(http.MethodGet, path, nil)
}

func (c *APIClient) Post(path string, body interface{}) ([]byte, error) {
	return c.Do(http.MethodPost, path, body)
}

func (c *APIClient) Put(path string, body interface{}) ([]byte, error) {
	return c.Do(http.MethodPut, path, body)
}

func (c *APIClient) Delete(path string) ([]byte, error) {
	return c.Do(http.MethodDelete, path, nil)
}

func (c *APIClient) DeleteWithBody(path string, body interface{}) ([]byte, error) {
	return c.Do(http.MethodDelete, path, body)
}
