package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type apiClient struct {
	baseURL string
	token   string
	http    http.Client
}

func (c *apiClient) init() {
	if c.http.Timeout == 0 {
		c.http.Timeout = 30 * time.Second
	}
}

func (c *apiClient) do(method, path string, body any) ([]byte, error) {
	c.init()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", errResp.Error, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *apiClient) get(path string) ([]byte, error) {
	return c.do(http.MethodGet, path, nil)
}

func (c *apiClient) post(path string, body any) ([]byte, error) {
	return c.do(http.MethodPost, path, body)
}

func (c *apiClient) put(path string, body any) ([]byte, error) {
	return c.do(http.MethodPut, path, body)
}

func (c *apiClient) delete(path string) ([]byte, error) {
	return c.do(http.MethodDelete, path, nil)
}

func printJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		// Not JSON, print raw
		fmt.Println(string(data))
		return nil
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
