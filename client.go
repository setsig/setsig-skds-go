package setsig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL    = "https://api.setsig.com"
	DefaultTimeout    = 15 * time.Second
	DefaultMaxRetries = 3
	SDKVersion        = "0.8.0"
	UserAgent         = "setsig-go/" + SDKVersion
)

// Option specifies configuration options for Client.
type Option func(*Client)

// WithBaseURL configures a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHTTPClient overrides the underlying net/http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout configures the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithMaxRetries sets the maximum retry attempts for network/rate limit errors.
func WithMaxRetries(retries int) Option {
	return func(c *Client) {
		if retries < 0 {
			retries = 0
		}
		c.maxRetries = retries
	}
}

// Client is the setsig Go SDK client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int

	Tokens   *TokensService
	Sessions *SessionsService
	Venues   *VenuesService
	Billing  *BillingService
	Webhooks *WebhooksService
	Terminals *TerminalsService
}

// NewClient initializes a new setsig client with the provided API key and optional configurations.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	c := &Client{
		baseURL:    DefaultBaseURL,
		apiKey:     apiKey,
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.timeout,
		}
	}

	c.Tokens = &TokensService{client: c}
	c.Sessions = &SessionsService{client: c}
	c.Venues = &VenuesService{client: c}
	c.Billing = &BillingService{client: c}
	c.Webhooks = &WebhooksService{client: c}
	c.Terminals = &TerminalsService{client: c}

	return c, nil
}

// RequestOptions configures per-request parameters like idempotency key.
type RequestOptions struct {
	IdempotencyKey string
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, target any, reqOpts ...RequestOptions) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}

	var idempotencyKey string
	if len(reqOpts) > 0 && reqOpts[0].IdempotencyKey != "" {
		idempotencyKey = reqOpts[0].IdempotencyKey
	}

	var lastErr error
	backoff := 200 * time.Millisecond

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff + jitter):
				backoff *= 2
			}
		}

		var reader io.Reader
		if len(reqBodyBytes) > 0 {
			reader = bytes.NewReader(reqBodyBytes)
		}

		httpReq, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("User-Agent", UserAgent)
		if idempotencyKey != "" {
			httpReq.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			continue // network transient failure, retry
		}

		respBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response body: %w", err)
			continue
		}

		// Handle Successful Responses (2xx)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if target != nil && len(respBytes) > 0 {
				if err := json.Unmarshal(respBytes, target); err != nil {
					return fmt.Errorf("decode response: %w", err)
				}
			}
			return nil
		}

		// Parse API error envelope
		var errEnv struct {
			Error   string      `json:"error"`
			Message string      `json:"message"`
			Code    string      `json:"code"`
			Details interface{} `json:"details,omitempty"`
		}
		_ = json.Unmarshal(respBytes, &errEnv)
		errMsg := errEnv.Message
		if errMsg == "" {
			errMsg = errEnv.Error
		}
		if errMsg == "" {
			errMsg = http.StatusText(resp.StatusCode)
		}

		baseErr := APIError{
			StatusCode: resp.StatusCode,
			Code:       errEnv.Code,
			Message:    errMsg,
		}

		// Handle Rate Limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			var retryAfter int
			if h := resp.Header.Get("Retry-After"); h != "" {
				retryAfter, _ = strconv.Atoi(h)
			}
			rateErr := &RateLimitError{
				APIError:          baseErr,
				RetryAfterSeconds: retryAfter,
			}
			lastErr = rateErr
			if attempt < c.maxRetries {
				if retryAfter > 0 {
					backoff = time.Duration(retryAfter) * time.Second
				}
				continue
			}
			return rateErr
		}

		// Handle Server Errors (5xx)
		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			lastErr = &baseErr
			continue
		}

		// Map status codes and error codes to specific client errors
		switch {
		case errEnv.Code == "ERR_PAYWALL_BLOCKED":
			var paywallURL string
			var curBal, reqDeposit float64
			var curr string
			if d, ok := errEnv.Details.(map[string]interface{}); ok {
				if u, ok := d["paywall_url"].(string); ok {
					paywallURL = u
				}
				if b, ok := d["current_balance"].(float64); ok {
					curBal = b
				}
				if r, ok := d["required_min_topup"].(float64); ok {
					reqDeposit = r
				}
				if c, ok := d["currency"].(string); ok {
					curr = c
				}
			}
			return &PaywallBlockedError{
				APIError:         baseErr,
				PaywallURL:       paywallURL,
				CurrentBalance:   curBal,
				RequiredMinTopup: reqDeposit,
				Currency:         curr,
			}
		case errEnv.Code == "ERR_CROSS_BORDER_CREDIT_REQUIRED":
			return &CrossBorderCreditRequiredError{APIError: baseErr}
		case errEnv.Code == "ERR_PAYWALL_TIER_REQUIRED":
			return &PaywallTierRequiredError{APIError: baseErr}
		case errEnv.Code == "ERR_ATTESTATION_FAILED":
			return &AttestationFailedError{APIError: baseErr}
		case errEnv.Code == "ERR_VISUAL_PROOF_REQUIRED":
			return &VisualProofRequiredError{APIError: baseErr}
		case errEnv.Code == "ERR_STALE_VISUAL_PROOF":
			return &StaleVisualProofError{APIError: baseErr}
		case errEnv.Code == "ERR_INVALID_IMAGE_HASH":
			return &InvalidImageHashError{APIError: baseErr}
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return &AuthenticationError{APIError: baseErr}
		case resp.StatusCode == http.StatusPaymentRequired:
			return &InsufficientCreditsError{APIError: baseErr}
		case resp.StatusCode == http.StatusConflict:
			if errEnv.Code == "DOUBLE_SPEND_REPLAY_ATTACK" || errEnv.Code == "ERR_REPLAY_DETECTED" || strings.Contains(strings.ToLower(errMsg), "replay") {
				return &ReplayAttackError{APIError: baseErr}
			}
			return &baseErr
		case resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusBadRequest:
			if errEnv.Code == "ERR_GEOFENCE_OUT_OF_BOUNDS" || strings.Contains(strings.ToLower(errMsg), "geofence") || strings.Contains(strings.ToLower(errMsg), "distance") {
				return &GeofenceOutOfBoundsError{APIError: baseErr}
			}
			return &baseErr
		default:
			return &baseErr
		}
	}

	return lastErr
}

