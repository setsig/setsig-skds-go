package setsig_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/setsig/setsig-go"
)

func TestComputeSignature(t *testing.T) {
	secret := "whsec_test_secret_12345"
	payload := []byte(`{"id":"evt_1","event":"session.verified"}`)
	timestamp := int64(1700000000)

	sig1 := setsig.ComputeSignature(payload, timestamp, secret)
	sig2 := setsig.ComputeSignature(payload, timestamp, secret)

	if sig1 == "" {
		t.Fatal("expected non-empty signature")
	}
	if sig1 != sig2 {
		t.Fatalf("expected deterministic signature, got %s and %s", sig1, sig2)
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "whsec_super_secret"
	payload := []byte(`{"id":"evt_test_99","event":"session.verified"}`)
	now := time.Now().Unix()
	signature := setsig.ComputeSignature(payload, now, secret)
	header := fmt.Sprintf("t=%d,v1=%s", now, signature)

	err := setsig.VerifySignature(payload, header, secret, 300*time.Second)
	if err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	secret := "whsec_super_secret"
	payload := []byte(`{"id":"evt_test_99","event":"session.verified"}`)
	now := time.Now().Unix()
	header := fmt.Sprintf("t=%d,v1=tampered_signature_hex_value", now)

	err := setsig.VerifySignature(payload, header, secret, 300*time.Second)
	if err != setsig.ErrInvalidWebhookSignature {
		t.Fatalf("expected ErrInvalidWebhookSignature, got: %v", err)
	}
}

func TestVerifySignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_super_secret"
	payload := []byte(`{"id":"evt_test_99","event":"session.verified"}`)
	expiredTime := time.Now().Add(-10 * time.Minute).Unix()
	signature := setsig.ComputeSignature(payload, expiredTime, secret)
	header := fmt.Sprintf("t=%d,v1=%s", expiredTime, signature)

	err := setsig.VerifySignature(payload, header, secret, 300*time.Second)
	if err != setsig.ErrWebhookTimestampExpired {
		t.Fatalf("expected ErrWebhookTimestampExpired, got: %v", err)
	}
}

func TestVerifySignature_MalformedHeader(t *testing.T) {
	secret := "whsec_super_secret"
	payload := []byte(`{"id":"evt_test_99"}`)

	testCases := []string{
		"",
		"invalid-format",
		"t=12345",
		"v1=abcde",
	}

	for _, header := range testCases {
		err := setsig.VerifySignature(payload, header, secret, 300*time.Second)
		if err != setsig.ErrInvalidHeaderFormat {
			t.Fatalf("for header %q, expected ErrInvalidHeaderFormat, got: %v", header, err)
		}
	}
}

func TestConstructEvent(t *testing.T) {
	client, err := setsig.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to init client: %v", err)
	}

	secret := "whsec_event_parser"
	payload := []byte(`{"id":"evt_abc","event":"session.verified","timestamp":1700000000,"data":{"status":"VERIFIED"}}`)
	now := time.Now().Unix()
	signature := setsig.ComputeSignature(payload, now, secret)
	header := fmt.Sprintf("t=%d,v1=%s", now, signature)

	event, err := client.Webhooks.ConstructEvent(payload, header, secret)
	if err != nil {
		t.Fatalf("expected successful event parsing, got: %v", err)
	}

	if event.ID != "evt_abc" || event.Event != "session.verified" {
		t.Fatalf("unexpected event data: %+v", event)
	}
}

func TestWebhookHandler(t *testing.T) {
	secret := "whsec_handler_test"
	var receivedEvent *setsig.WebhookEvent

	handler := setsig.WebhookHandler(secret, func(w http.ResponseWriter, r *http.Request, event *setsig.WebhookEvent) {
		receivedEvent = event
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	})

	payload := []byte(`{"id":"evt_handled_1","event":"session.verified","timestamp":1700000000}`)
	now := time.Now().Unix()
	sig := setsig.ComputeSignature(payload, now, secret)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(payload))
	req.Header.Set("X-Setsig-Signature", header)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	if receivedEvent == nil || receivedEvent.ID != "evt_handled_1" {
		t.Fatalf("expected handler to process event, got: %+v", receivedEvent)
	}
}

