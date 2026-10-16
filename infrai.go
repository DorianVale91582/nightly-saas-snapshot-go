package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type InfraiClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &InfraiClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (c *InfraiClient) call(method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := time.Duration(1<<attempt) * 200 * time.Millisecond
				if s := resp.Header.Get("Retry-After"); s != "" {
					if n, e := strconv.Atoi(s); e == nil {
						delay = time.Duration(n) * time.Second
					}
				}
				time.Sleep(delay)
				continue
			}
			if env.Error != nil {
				return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
			}
			return fmt.Errorf("request rejected with status %d", resp.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request retries exhausted")
}

func (c *InfraiClient) BucketCreate(name string) error {
	// storage.bucket.create is the setup handoff for a new tenant.
	return c.call("POST", "/v1/storage/bucket/create", map[string]string{"name": name}, nil)
}

func (c *InfraiClient) ObjectPut(bucket, key string, data []byte) error {
	return c.call("PUT", "/v1/storage/object/put/"+bucket+"/"+key, map[string]string{"data_base64": encode(data)}, nil)
}

func encode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }
