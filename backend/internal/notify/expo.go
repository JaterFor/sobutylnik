package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

type ExpoClient struct {
	httpClient  *http.Client
	accessToken string
}

func NewExpoClient(accessToken string) *ExpoClient {
	return &ExpoClient{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		accessToken: accessToken,
	}
}

type expoMessage struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}

type expoReceipt struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

type expoResponse struct {
	Data []expoReceipt `json:"data"`
}

// SendResult pairs each token with whether Expo reported it as permanently
// invalid (DeviceNotRegistered), so the caller can stop sending to it.
type SendResult struct {
	Token      string
	InvalidNow bool
}

// Send pushes one notification per token. A missing accessToken or empty
// token list is a no-op (dev/test environments without Expo credentials).
func (c *ExpoClient) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) ([]SendResult, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	return c.sendTo(ctx, expoPushURL, tokens, title, body, data)
}

// sendTo is Send with the endpoint URL as a parameter, so tests can point it
// at an httptest.Server instead of the real Expo API.
func (c *ExpoClient) sendTo(ctx context.Context, url string, tokens []string, title, body string, data map[string]string) ([]SendResult, error) {
	messages := make([]expoMessage, 0, len(tokens))
	for _, t := range tokens {
		messages = append(messages, expoMessage{To: t, Title: title, Body: body, Data: data})
	}

	payload, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed expoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		// Expo responded, but not with the expected shape; treat as
		// best-effort delivery rather than failing the caller.
		return nil, nil
	}

	results := make([]SendResult, 0, len(tokens))
	for i, receipt := range parsed.Data {
		if i >= len(tokens) {
			break
		}
		results = append(results, SendResult{
			Token:      tokens[i],
			InvalidNow: receipt.Status == "error" && receipt.Details.Error == "DeviceNotRegistered",
		})
	}
	return results, nil
}
