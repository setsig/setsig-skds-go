package setsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultWebhookTolerance is the default allowable drift for webhook delivery timestamps (5 minutes).
	DefaultWebhookTolerance = 300 * time.Second
)

// WebhooksService manages incoming event verification and handling.
type WebhooksService struct {
	client *Client
}

// ComputeSignature calculates the HMAC-SHA256 hex digest for a given payload and timestamp.
func ComputeSignature(payload []byte, timestamp int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	signedPayload := fmt.Sprintf("t=%d.%s", timestamp, string(payload))
	mac.Write([]byte(signedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

// ParseHeader parses the X-Setsig-Signature header into its timestamp and signature components.
func ParseHeader(header string) (int64, string, error) {
	if header == "" {
		return 0, "", ErrInvalidHeaderFormat
	}

	parts := strings.Split(header, ",")
	var timestampStr, signature string

	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			switch kv[0] {
			case "t":
				timestampStr = kv[1]
			case "v1":
				signature = kv[1]
			}
		}
	}

	if timestampStr == "" || signature == "" {
		return 0, "", ErrInvalidHeaderFormat
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0, "", ErrInvalidHeaderFormat
	}

	return timestamp, signature, nil
}

// VerifySignature validates that an incoming webhook payload was authentically signed by setsig.
// ConstantTimeCompare is strictly utilized to eliminate timing attacks.
func VerifySignature(payload []byte, header string, secret string, tolerance time.Duration) error {
	timestamp, receivedSig, err := ParseHeader(header)
	if err != nil {
		return err
	}

	if tolerance <= 0 {
		tolerance = DefaultWebhookTolerance
	}

	now := time.Now().Unix()
	if now-timestamp > int64(tolerance.Seconds()) || timestamp-now > int64(tolerance.Seconds()) {
		return ErrWebhookTimestampExpired
	}

	expectedSig := ComputeSignature(payload, timestamp, secret)

	expectedSigBytes := []byte(expectedSig)
	receivedSigBytes := []byte(receivedSig)

	if subtle.ConstantTimeCompare(expectedSigBytes, receivedSigBytes) != 1 {
		return ErrInvalidWebhookSignature
	}

	return nil
}

// ConstructEvent parses and cryptographically validates an incoming setsig webhook event.
func (s *WebhooksService) ConstructEvent(payload []byte, header string, secret string, tolerance ...time.Duration) (*WebhookEvent, error) {
	tol := DefaultWebhookTolerance
	if len(tolerance) > 0 {
		tol = tolerance[0]
	}

	if err := VerifySignature(payload, header, secret, tol); err != nil {
		return nil, err
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("setsig: failed to unmarshal webhook payload: %w", err)
	}

	return &event, nil
}

// HandlerFunc is the callback signature for processed setsig webhook events.
type HandlerFunc func(w http.ResponseWriter, r *http.Request, event *WebhookEvent)

// WebhookHandler returns a standard http.HandlerFunc that verifies incoming webhook requests
// before passing the verified WebhookEvent to your handler.
func WebhookHandler(secret string, fn HandlerFunc, tolerance ...time.Duration) http.HandlerFunc {
	tol := DefaultWebhookTolerance
	if len(tolerance) > 0 {
		tol = tolerance[0]
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		header := r.Header.Get("X-Setsig-Signature")
		if header == "" {
			http.Error(w, "Missing X-Setsig-Signature Header", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if err := VerifySignature(body, header, secret, tol); err != nil {
			if errorsIs(err, ErrWebhookTimestampExpired) {
				http.Error(w, "Webhook timestamp expired", http.StatusBadRequest)
				return
			}
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		var event WebhookEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "Invalid payload JSON", http.StatusBadRequest)
			return
		}

		fn(w, r, &event)
	}
}

func errorsIs(err, target error) bool {
	return err == target
}

