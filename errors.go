package setsig

import (
	"errors"
	"fmt"
)

var (
	// ErrMissingAPIKey is returned when the client is constructed without an API key.
	ErrMissingAPIKey = errors.New("setsig: api key is required")

	// ErrInvalidWebhookSignature is returned when a webhook signature fails validation.
	ErrInvalidWebhookSignature = errors.New("setsig: invalid webhook signature")

	// ErrWebhookTimestampExpired is returned when a webhook timestamp exceeds the tolerance window.
	ErrWebhookTimestampExpired = errors.New("setsig: webhook timestamp outside acceptable tolerance window")

	// ErrInvalidHeaderFormat is returned when the signature header format is malformed.
	ErrInvalidHeaderFormat = errors.New("setsig: malformed signature header format, expected t=timestamp,v1=signature")
)

// APIError represents an error returned by the setsig API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("setsig api error (HTTP %d - %s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("setsig api error (HTTP %d): %s", e.StatusCode, e.Message)
}

// AuthenticationError indicates an invalid or missing API key.
type AuthenticationError struct {
	APIError
}

// InsufficientCreditsError indicates balance depletion on account.
type InsufficientCreditsError struct {
	APIError
}

// ReplayAttackError indicates a double-spend attempt or reused client token.
type ReplayAttackError struct {
	APIError
}

// GeofenceOutOfBoundsError indicates verification failed because coordinates were outside venue perimeter.
type GeofenceOutOfBoundsError struct {
	APIError
}

// RateLimitError indicates request throttling.
type RateLimitError struct {
	APIError
	RetryAfterSeconds int
}

func (e *RateLimitError) Error() string {
	if e.RetryAfterSeconds > 0 {
		return fmt.Sprintf("setsig rate limit exceeded (HTTP %d): retry after %ds", e.StatusCode, e.RetryAfterSeconds)
	}
	return fmt.Sprintf("setsig rate limit exceeded (HTTP %d): %s", e.StatusCode, e.Message)
}

