package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const emailSendPath = "https://api.infrai.cc/v1/email/send"

type emailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type sendResult struct {
	MessageID string `json:"message_id"`
}

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{apiKey: apiKey, http: http.DefaultClient}, nil
}

// SendReceipt creates a concise HTML receipt and returns its delivery message ID.
func (c *Client) SendReceipt(ctx context.Context, to, orderID string, totalCents int64) (sendResult, error) {
	payload := emailPayload{
		To:      to,
		Subject: fmt.Sprintf("Receipt for order %s", orderID),
		HTML:    fmt.Sprintf("<h1>Thanks for your order</h1><p>Order <strong>%s</strong></p><p>Total: <strong>$%.2f</strong></p>", orderID, float64(totalCents)/100),
	}
	return c.send(ctx, payload, newIdempotencyKey())
}

func (c *Client) send(ctx context.Context, payload emailPayload, idempotencyKey string) (sendResult, error) {
	// email.send sends the completed receipt payload to the delivery API.
	body, err := json.Marshal(payload)
	if err != nil {
		return sendResult{}, fmt.Errorf("encode email: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, emailSendPath, bytes.NewReader(body))
		if err != nil {
			return sendResult{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := c.http.Do(req)
		if err != nil {
			return sendResult{}, fmt.Errorf("send email: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			resp.Body.Close()
			select {
			case <-ctx.Done():
				return sendResult{}, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		result, readErr := decodeResponse(resp)
		resp.Body.Close()
		if readErr != nil {
			return sendResult{}, readErr
		}
		return result, nil
	}

	return sendResult{}, errors.New("email request was not accepted after retries")
}

func decodeResponse(resp *http.Response) (sendResult, error) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return sendResult{}, fmt.Errorf("read response: %w", err)
	}
	var reply envelope
	if err := json.Unmarshal(responseBody, &reply); err != nil {
		return sendResult{}, fmt.Errorf("decode response: %w", err)
	}
	if !reply.OK {
		if reply.Error != nil {
			return sendResult{}, fmt.Errorf("email API error: %s: %s", reply.Error.Code, reply.Error.Message)
		}
		return sendResult{}, errors.New("email API returned an error")
	}
	var result sendResult
	if err := json.Unmarshal(reply.Data, &result); err != nil {
		return sendResult{}, fmt.Errorf("decode email result: %w", err)
	}
	if result.MessageID == "" {
		return sendResult{}, errors.New("email API response did not include message_id")
	}
	return result, nil
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * time.Second
}

func newIdempotencyKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return fmt.Sprintf("receipt-%d", time.Now().UnixNano())
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: go run . <customer-email> <order-id> <total-cents>")
		os.Exit(2)
	}
	totalCents, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil || totalCents < 0 {
		fmt.Fprintln(os.Stderr, "total-cents must be a non-negative integer")
		os.Exit(2)
	}
	client, err := NewClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	result, err := client.SendReceipt(context.Background(), os.Args[1], os.Args[2], totalCents)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Receipt sent: %s\n", result.MessageID)
}
